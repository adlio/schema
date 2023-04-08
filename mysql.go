package schema

import (
	"context"
	"fmt"
	"hash/crc32"
	"strings"
	"time"
)

const mysqlLockSalt uint32 = 271192482

// MySQL is the dialect which should be used for MySQL/MariaDB databases
var MySQL = mysqlDialect{}

type mysqlDialect struct{}

// Lock implements the Locker interface to obtain a global lock before the
// migrations are run.
func (m mysqlDialect) Lock(ctx context.Context, tx Queryer, tableName string) error {
	lockID := m.advisoryLockID(tableName)
	query := fmt.Sprintf(`SELECT GET_LOCK('%s', 10)`, lockID)
	_, err := tx.ExecContext(ctx, query)
	return err
}

// Unlock implements the Locker interface to release the global lock after the
// migrations are run.
func (m mysqlDialect) Unlock(ctx context.Context, tx Queryer, tableName string) error {
	lockID := m.advisoryLockID(tableName)
	query := fmt.Sprintf(`SELECT RELEASE_LOCK('%s')`, lockID)
	_, err := tx.ExecContext(ctx, query)
	return err
}

// CreateMigrationsTable implements the Dialect interface to create the
// table which tracks applied migrations. It only creates the table if it
// does not already exist
func (m mysqlDialect) CreateMigrationsTable(ctx context.Context, tx Queryer, tableName string) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id VARCHAR(255) NOT NULL,
			checksum VARCHAR(32) NOT NULL DEFAULT '',
			execution_time_in_millis INTEGER NOT NULL DEFAULT 0,
			applied_at TIMESTAMP NOT NULL
		)`, tableName)
	_, err := tx.ExecContext(ctx, query)
	return err
}

// InsertAppliedMigration implements the Dialect interface to insert a record
// into the migrations tracking table *after* a migration has successfully
// run.
func (m mysqlDialect) InsertAppliedMigration(ctx context.Context, tx Queryer, tableName string, am *AppliedMigration) error {
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
func (m mysqlDialect) GetAppliedMigrations(ctx context.Context, tx Queryer, tableName string) (migrations []*AppliedMigration, err error) {
	migrations = make([]*AppliedMigration, 0)

	query := fmt.Sprintf(`
		SELECT id, checksum, execution_time_in_millis, applied_at
		FROM %s
		ORDER BY id ASC`, tableName)
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return migrations, err
	}
	defer rows.Close()

	for rows.Next() {
		migration := AppliedMigration{}

		var appliedAt mysqlTime
		err = rows.Scan(&migration.ID, &migration.Checksum, &migration.ExecutionTimeInMillis, &appliedAt)
		if err != nil {
			err = fmt.Errorf("Failed to GetAppliedMigrations. Did somebody change the structure of the %s table?: %w", tableName, err)
			return migrations, err
		}
		migration.AppliedAt = appliedAt.Value
		migrations = append(migrations, &migration)
	}

	return migrations, err
}

// QuotedTableName returns the string value of the name of the migration
// tracking table after it has been quoted for MySQL
func (m mysqlDialect) QuotedTableName(schemaName, tableName string) string {
	if schemaName == "" {
		return m.quotedIdent(tableName)
	}
	return m.quotedIdent(schemaName) + "." + m.quotedIdent(tableName)
}

// quotedIdent wraps the supplied string in the MySQL identifier
// quote character
func (m mysqlDialect) quotedIdent(ident string) string {
	if ident == "" {
		return ""
	}
	return "`" + strings.ReplaceAll(ident, "`", "``") + "`"
}

// advisoryLockID generates a table-specific lock name to use
func (m mysqlDialect) advisoryLockID(tableName string) string {
	sum := crc32.ChecksumIEEE([]byte(tableName))
	sum = sum * mysqlLockSalt
	return fmt.Sprint(sum)
}

type mysqlTime struct {
	Value time.Time
}

func (t *mysqlTime) Scan(src interface{}) (err error) {
	if src == nil {
		t.Value = time.Time{}
	}

	if srcTime, isTime := src.(time.Time); isTime {
		t.Value = srcTime.In(time.Local)
		return nil
	}

	return t.ScanString(fmt.Sprintf("%s", src))
}

func (t *mysqlTime) ScanString(src string) (err error) {
	switch len(src) {
	case 19:
		t.Value, err = time.ParseInLocation("2006-01-02 15:04:05", src, time.UTC)
		if err != nil {
			return err
		}
	}
	t.Value = t.Value.In(time.Local)
	return nil
}
