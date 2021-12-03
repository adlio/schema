package schema

import (
	"fmt"
	"hash/crc32"
	"strings"
	"unicode"
)

const postgresAdvisoryLockSalt uint32 = 542384964

// Postgres is the dialect for Postgres-compatible
// databases
var Postgres = postgresDialect{}

type postgresDialect struct{}

func (p postgresDialect) Lock(db Connection, tableName string) error {
	lockID := p.advisoryLockID(tableName)
	_, err := db.Exec(`SELECT pg_advisory_lock($1)`, lockID)
	return err
}

func (p postgresDialect) Unlock(db Connection, tableName string) error {
	lockID := p.advisoryLockID(tableName)
	_, err := db.Exec(`SELECT pg_advisory_unlock($1)`, lockID)
	return err
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
// returns the SQL statement to retrieve all records from it
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
		return p.QuotedIdent(tableName)
	}
	return p.QuotedIdent(schemaName) + "." + p.QuotedIdent(tableName)
}

// QuotedIdent wraps the supplied string in the Postgres identifier
// quote character
func (p postgresDialect) QuotedIdent(ident string) string {
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

// advisoryLockID generates a table-specific lock name to use
func (p postgresDialect) advisoryLockID(tableName string) string {
	sum := crc32.ChecksumIEEE([]byte(tableName))
	sum = sum * postgresAdvisoryLockSalt
	return fmt.Sprint(sum)
}
