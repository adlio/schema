package schema

import (
	"context"
	"fmt"
	"hash/crc32"
	"strings"
	"time"
	"unicode"
)

const mssqlAdvisoryLockSalt uint32 = 542384964

// MSSQL is the dialect for MS SQL-compatible databases
var MSSQL = mssqlDialect{}

type mssqlDialect struct{}

// Lock implements the Locker interface to obtain a global lock before the
// migrations are run. It uses SQL Server's sp_getapplock stored procedure
// with a session-based lock to ensure that only one process can run migrations
// at a time, which is critical for clustered environments.
func (s mssqlDialect) Lock(ctx context.Context, tx Queryer, tableName string) error {
	lockID := s.advisoryLockID(tableName)
	// Use application lock without explicit transaction
	query := fmt.Sprintf("EXEC sp_getapplock @Resource = '%d', @LockMode = 'Exclusive', @LockOwner = 'Session';", lockID)
	_, err := tx.ExecContext(ctx, query)
	return err
}

// Unlock implements the Locker interface to release the global lock after the
// migrations are run. It first checks if we have the lock before trying to
// release it to avoid errors when the lock is not held.
func (s mssqlDialect) Unlock(ctx context.Context, tx Queryer, tableName string) error {
	lockID := s.advisoryLockID(tableName)
	
	// First check if we have the lock before trying to release it
	checkQuery := fmt.Sprintf("SELECT APPLOCK_MODE('public', '%d', 'Session');", lockID)
	rows, err := tx.QueryContext(ctx, checkQuery)
	if err != nil {
		// If there was an error checking, just return success
		return nil
	}
	defer rows.Close()
	
	// Check if we have the lock
	var lockMode string
	if rows.Next() {
		err = rows.Scan(&lockMode)
		if err != nil || lockMode == "NoLock" {
			// If we don't have the lock, just return success
			return nil
		}
	}
	
	// Release the application lock
	query := fmt.Sprintf("EXEC sp_releaseapplock @Resource = '%d', @LockOwner = 'Session';", lockID)
	_, err = tx.ExecContext(ctx, query)
	return err
}

// advisoryLockID generates a consistent integer ID for use with SQL Server's sp_getapplock
// based on the table name. It uses a CRC32 checksum of the table name XORed with a salt
// to ensure uniqueness across different applications using the same database.
func (s mssqlDialect) advisoryLockID(tableName string) uint32 {
	return crc32.ChecksumIEEE([]byte(tableName)) ^ mssqlAdvisoryLockSalt
}


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
	
	// Handle concurrent table creation: ignore "object already exists" errors
	if err != nil && strings.Contains(err.Error(), "There is already an object named") {
		return nil
	}
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
