package schema

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"
)

// MSSQL is the dialect for MS SQL-compatible databases
var MSSQL = mssqlDialect{}

type mssqlDialect struct{}

func (s mssqlDialect) QuotedTableName(schemaName, tableName string) string {
	if schemaName == "" {
		return s.QuotedIdent(tableName)
	}
	return fmt.Sprintf("%s.%s", s.QuotedIdent(schemaName), s.QuotedIdent(tableName))
}

func (s mssqlDialect) QuotedIdent(ident string) string {
	if ident == "" {
		return ""
	}

	var sb strings.Builder
	sb.WriteRune('[')
	for _, r := range ident {
		switch {
		case unicode.IsSpace(r):
			continue
		case r == ';':
			continue
		case r == ']':
			sb.WriteRune(r)
			sb.WriteRune(r)
		default:
			sb.WriteRune(r)
		}
	}
	sb.WriteRune(']')

	return sb.String()
}

func (s mssqlDialect) CreateMigrationsTable(ctx context.Context, tx Queryer, tableName string) error {
	unquotedTableName := tableName[1 : len(tableName)-1]
	query := fmt.Sprintf(`
		IF NOT EXISTS (SELECT * FROM Sysobjects WHERE NAME='%s' AND XTYPE='U')
			CREATE TABLE %s (
				id VARCHAR(255) NOT NULL,
				checksum VARCHAR(32) NOT NULL DEFAULT '',
				execution_time_in_millis INTEGER NOT NULL DEFAULT 0,
				applied_at DATETIMEOFFSET NOT NULL
			)
	`, unquotedTableName, tableName)
	_, err := tx.ExecContext(ctx, query)
	return err
}

func (s mssqlDialect) GetAppliedMigrations(ctx context.Context, tx Queryer, tableName string) (migrations []*AppliedMigration, err error) {
	migrations = make([]*AppliedMigration, 0)

	query := fmt.Sprintf(`
		SELECT id, checksum, execution_time_in_millis, applied_at
		FROM %s ORDER BY id ASC
	`, tableName)

	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return migrations, err
	}
	defer rows.Close()

	for rows.Next() {
		migration := AppliedMigration{}
		err = rows.Scan(&migration.ID, &migration.Checksum, &migration.ExecutionTimeInMillis, &migration.AppliedAt)
		if err != nil {
			err = fmt.Errorf("failed to GetAppliedMigrations. Did somebody change the structure of the %s table?: %w", tableName, err)
			return migrations, err
		}
		migration.AppliedAt = migration.AppliedAt.In(time.Local)
		migrations = append(migrations, &migration)
	}

	return migrations, err
}

func (s mssqlDialect) InsertAppliedMigration(ctx context.Context, tx Queryer, tableName string, am *AppliedMigration) error {
	query := fmt.Sprintf(`
		INSERT INTO %s
		( id, checksum, execution_time_in_millis, applied_at )
		VALUES
		( @p1, @p2, @p3, @p4 )`,
		tableName,
	)
	_, err := tx.ExecContext(ctx, query, am.ID, am.MD5(), am.ExecutionTimeInMillis, am.AppliedAt)
	return err
}
