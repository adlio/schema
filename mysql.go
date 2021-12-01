package schema

import (
	"fmt"
	"hash/crc32"
	"strings"
)

const mysqlLockSalt uint32 = 271192482

var MySQL = mysqlDialect{}

type mysqlDialect struct{}

func (m mysqlDialect) Lock(db Queryer, tableName string) error {
	lockID := m.advisoryLockID(tableName)
	_, err := db.Exec(`SELECT GET_LOCK(?, 10)`, lockID)
	return err
}

func (m mysqlDialect) Unlock(db Queryer, tableName string) error {
	lockID := m.advisoryLockID(tableName)
	_, err := db.Exec(`SELECT RELEASE_LOCK(?)`, lockID)
	return err
}

func (m mysqlDialect) CreateSQL(tableName string) string {
	return fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id VARCHAR(255) NOT NULL,
			checksum VARCHAR(32) NOT NULL DEFAULT '',
			execution_time_in_millis INTEGER NOT NULL DEFAULT 0,
			applied_at TIMESTAMP NOT NULL
		)`, tableName)
}

func (m mysqlDialect) InsertSQL(tableName string) string {
	return fmt.Sprintf(`
		INSERT INTO %s
		( id, checksum, execution_time_in_millis, applied_at )
		VALUES
		( ?, ?, ?, ? )
		`, tableName)
}

func (m mysqlDialect) SelectSQL(tableName string) string {
	return fmt.Sprintf(`
		SELECT id, checksum, execution_time_in_millis, applied_at
		FROM %s
		ORDER BY id ASC;
	`, tableName)
}

// QuotedTableName returns the string value of the name of the migration
// tracking table after it has been quoted for MySQL
//
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

type nullMySQLLogger struct{}

func (nsl nullMySQLLogger) Print(v ...interface{}) {
	// Intentional no-op. The purpose of this class is to swallow/ignore
	// the MySQL driver errors which occur while we're waiting for the Docker
	// MySQL instance to start up.
}
