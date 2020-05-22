package schema

import (
	"fmt"
	"hash/crc32"
	"strings"
)

const postgresAdvisoryLockSalt uint32 = 542384964

// Postgres is the dialect for Postgres-compatible
// databases
var Postgres = postgresDialect{}

// Postgres is the Postgresql dialect
type postgresDialect struct{}

func (p postgresDialect) LockSQL(tableName string) string {
	lockID := p.advisoryLockID(tableName)
	return fmt.Sprintf(`SELECT pg_advisory_lock(%s)`, lockID)
}

func (p postgresDialect) UnlockSQL(tableName string) string {
	lockID := p.advisoryLockID(tableName)
	return fmt.Sprintf(`SELECT pg_advisory_unlock(%s)`, lockID)
}

// CreateSQL takes the name of the migration tracking table and
// returns the SQL statement needed to create it
func (p postgresDialect) CreateSQL(tableName string) string {
	return fmt.Sprintf(`
				CREATE TABLE IF NOT EXISTS %s (
					id VARCHAR(255) NOT NULL,
					checksum VARCHAR(32) NOT NULL DEFAULT '',
					execution_time_in_millis INTEGER NOT NULL DEFAULT 0,
					applied_at TIMESTAMP WITH TIME ZONE NOT NULL
				)
			`, tableName)
}

// InsertSQL takes the name of the migration tracking table and
// returns the SQL statement needed to insert a migration into it
func (p postgresDialect) InsertSQL(tableName string) string {
	return fmt.Sprintf(`
				INSERT INTO %s
				( id, checksum, execution_time_in_millis, applied_at )
				VALUES
				( $1, $2, $3, $4 )
				`,
		tableName,
	)
}

// SelectSQL takes the name of the migration tracking table and
// returns trhe SQL statement to retrieve all records from it
//
func (p postgresDialect) SelectSQL(tableName string) string {
	return fmt.Sprintf(`
		SELECT id, checksum, execution_time_in_millis, applied_at
		FROM %s
		ORDER BY id ASC
	`, tableName)
}

// QuotedTableName returns the string value of the name of the migration
// tracking table after it has been quoted for Postgres
//
func (p postgresDialect) QuotedTableName(schemaName, tableName string) string {
	if schemaName == "" {
		return p.quotedIdent(tableName)
	}
	return p.quotedIdent(schemaName) + "." + p.quotedIdent(tableName)
}

// quotedIdent wraps the supplied string in the Postgres identifier
// quote character
func (p postgresDialect) quotedIdent(ident string) string {
	return `"` + strings.ReplaceAll(ident, `"`, "") + `"`
}

// advisoryLockID generates a table-specific lock name to use
func (p postgresDialect) advisoryLockID(tableName string) string {
	sum := crc32.ChecksumIEEE([]byte(tableName))
	sum = sum * postgresAdvisoryLockSalt
	return fmt.Sprint(sum)
}
