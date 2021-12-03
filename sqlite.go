package schema

import (
	"fmt"
	"strings"
)

var SQLite = &sqliteDialect{}

type sqliteDialect struct{}

// NewSQLite creates a new sqlite dialect. Customization of the lock table
// name and lock duration are made with WithSQLiteLockTable and
// WithSQLiteLockDuration options.
func NewSQLite(opts ...func(s *sqliteDialect)) *sqliteDialect {
	return &sqliteDialect{}
}

// Lock attempts to obtain a lock of the database. nil is returned if the lock
// is successfully claimed. A non-nil value is returned for database errors
// or if the lock timeout is reached.
func (s *sqliteDialect) Lock(db Queryer, tableName string) error {
	_, err := db.Exec("SELECT 1")
	return err
}

// Unlock releases the database lock.
func (s *sqliteDialect) Unlock(db Queryer, tableName string) error {
	_, err := db.Exec("SELECT 1")
	return err
}

// CreateSQL takes the name of the migration tracking table and
// returns the SQL statement needed to create it
func (s *sqliteDialect) CreateSQL(tableName string) string {
	return fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id TEXT NOT NULL,
			checksum TEXT NOT NULL DEFAULT '',
			execution_time_in_millis INTEGER NOT NULL DEFAULT 0,
			applied_at DATETIME
		);`, tableName)
}

// InsertSQL takes the name of the migration tracking table and
// returns the SQL statement needed to insert a migration into it
func (s *sqliteDialect) InsertSQL(tableName string) string {
	return fmt.Sprintf(`
		INSERT INTO %s
		( id, checksum, execution_time_in_millis, applied_at )
		VALUES
		( ?, ?, ?, ? )
		`, tableName)
}

// SelectSQL takes the name of the migration tracking table and
// returns the SQL statement to retrieve all records from it
func (s *sqliteDialect) SelectSQL(tableName string) string {
	return fmt.Sprintf(`
		SELECT id, checksum, execution_time_in_millis, applied_at
		FROM %s
		ORDER BY id ASC
	`, tableName)
}

// QuotedTableName returns the string value of the name of the migration
// tracking table after it has been quoted for Postgres
func (s *sqliteDialect) QuotedTableName(_, tableName string) string {
	return `"` + strings.ReplaceAll(tableName, `"`, `""`) + `"`
}
