package schema

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

const lockMagicNum = 794774819
const defaultSQLiteLockTable = "schema_lock"
const defaultLockDuration = 30 * time.Second

type sqliteDialect struct {
	mutex        sync.Mutex
	lockDuration time.Duration
	lockTable    string
	code         int64
}

var ErrSQLiteLockTimeout = errors.New("sqlite: timeout requesting lock")

// NewSQLite creates a new sqlite dialect. Customization of the lock table
// name and lock duration are made with WithSQLiteLockTable and
// WithSQLiteLockDuration options.
func NewSQLite(opts ...func(s *sqliteDialect)) *sqliteDialect {
	s := &sqliteDialect{
		lockDuration: defaultLockDuration,
		lockTable:    defaultSQLiteLockTable,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// WithSQLiteLockTable configures the lock table name. The default name
// without this option is 'schema_lock'.
func WithSQLiteLockTable(name string) func(s *sqliteDialect) {
	return func(s *sqliteDialect) {
		s.lockTable = name
	}
}

// WithSQLiteLockDuration sets the lock timeout and expiration. The default
// is 30 seconds. If the migration will take longer (e.g. copying of entire
// large tables), increase the timeout accordingly.
func WithSQLiteLockDuration(d time.Duration) func(s *sqliteDialect) {
	return func(s *sqliteDialect) {
		s.lockDuration = d
	}
}

// Lock attempts to obtain a lock of the database. nil is returned if the lock
// is successfully claimed. A non-nil value is returned for database errors
// or if the lock timeout is reached.
func (s *sqliteDialect) Lock(db Connection, tableName string) error {
	s.mutex.Lock()

	_, err := db.Exec(fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id INTEGER PRIMARY KEY,
			code INTEGER,
			expiration DATETIME NOT NULL)`, s.lockTable))
	if err != nil {
		return err
	}

	// Only try to fetch the lock for a limited time
	timeout := time.Now().Add(s.lockDuration)

	for time.Now().Before(timeout) {

		// Delete any expired locks
		_, err := db.Exec(
			fmt.Sprintf(`
				DELETE FROM %s
				WHERE datetime(expiration) < datetime('now')`, s.lockTable))
		if err != nil {
			return err
		}

		// Unique code to identify this lock during unlock
		code := time.Now().UnixNano()

		// Locking relies on the PRIMARY KEY constraint. Successfully inserting the id lockMagicNum
		// means the lock was obtained. An UNIQUE constraint error results in us trying again one
		// second later. Any other error is returned.
		_, err = db.Exec(
			fmt.Sprintf(`INSERT INTO %s (id, code, expiration) VALUES(?, ?, ?)`, s.lockTable),
			lockMagicNum, code, time.Now().Add(s.lockDuration))

		if err == nil {
			s.code = code
			return nil
		}

		if !isConstraintError(err) {
			return err
		}

		time.Sleep(time.Second)
	}

	return ErrSQLiteLockTimeout
}

// Unlock releases the database lock.
func (s *sqliteDialect) Unlock(db Connection, tableName string) error {
	defer s.mutex.Unlock()

	// Delete only the lock we created by checking 'code'. This guards against the
	// edge case where another process has deleted our expired lock and grabbed
	// their own just before we process Unlock().
	_, err := db.Exec(
		fmt.Sprintf(`DELETE FROM %s WHERE id=? AND code=?;`, s.lockTable), lockMagicNum, s.code)

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
// returns trhe SQL statement to retrieve all records from it
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
	return `"` + strings.ReplaceAll(tableName, `"`, "") + `"`
}

// isConstraintError returns whether the error is likely a uniqueness
// constraint violation. The string version is tested instead of checking
// for a driver-specific error as these would bring in a dependency
// (usually requiring cgo) for all users of the library.
func isConstraintError(err error) bool {
	s := strings.ToLower(err.Error())

	return strings.Contains(s, "constraint") || strings.Contains(s, "unique")
}
