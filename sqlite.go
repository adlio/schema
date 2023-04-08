package schema

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"
)

// SQLite is the dialect for sqlite3 databases
var SQLite = &sqliteDialect{}

type sqliteDialect struct{}

// CreateMigrationsTable implements the Dialect interface to create the
// table which tracks applied migrations. It only creates the table if it
// does not already exist
func (s sqliteDialect) CreateMigrationsTable(ctx context.Context, tx Queryer, tableName string) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id TEXT NOT NULL,
			checksum TEXT NOT NULL DEFAULT '',
			execution_time_in_millis INTEGER NOT NULL DEFAULT 0,
			applied_at DATETIME NOT NULL
		)`, tableName)
	_, err := tx.ExecContext(ctx, query)
	return err
}

// InsertAppliedMigration implements the Dialect interface to insert a record
// into the migrations tracking table *after* a migration has successfully
// run.
func (s *sqliteDialect) InsertAppliedMigration(ctx context.Context, tx Queryer, tableName string, am *AppliedMigration) error {
	query := fmt.Sprintf(`
		INSERT INTO %s
		( id, checksum, execution_time_in_millis, applied_at )
		VALUES
		( ?, ?, ?, ? )
		`, tableName,
	)
	_, err := tx.ExecContext(ctx, query, am.ID, am.MD5(), am.ExecutionTimeInMillis, am.AppliedAt)
	return err
}

// GetAppliedMigrations retrieves all data from the migrations tracking table
func (s sqliteDialect) GetAppliedMigrations(ctx context.Context, tx Queryer, tableName string) (migrations []*AppliedMigration, err error) {
	migrations = make([]*AppliedMigration, 0)

	query := fmt.Sprintf(`
		SELECT id, checksum, execution_time_in_millis, applied_at
		FROM %s
		ORDER BY id ASC
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
			err = fmt.Errorf("Failed to GetAppliedMigrations. Did somebody change the structure of the %s table?: %w", tableName, err)
			return migrations, err
		}
		migration.AppliedAt = migration.AppliedAt.In(time.Local)
		migrations = append(migrations, &migration)
	}

	return migrations, err
}

// QuotedTableName returns the string value of the name of the migration
// tracking table after it has been quoted for SQLite
func (s sqliteDialect) QuotedTableName(schemaName, tableName string) string {
	ident := schemaName + tableName
	if ident == "" {
		return ""
	}

	var sb strings.Builder
	sb.WriteRune('"')
	for _, r := range ident {
		switch {
		case unicode.IsSpace(r):
			// Skip spaces
			continue
		case r == '"':
			// Escape double-quotes with repeated double-quotes
			sb.WriteString(`""`)
		case r == ';':
			// Ignore the command termination character
			continue
		default:
			sb.WriteRune(r)
		}
	}
	sb.WriteRune('"')
	return sb.String()

}
