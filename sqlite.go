package schema

import (
	"fmt"
	"strings"
	"unicode"
)

var SQLite = &sqliteDialect{}

type sqliteDialect struct{}

// CreateSQL takes the name of the migration tracking table and
// returns the SQL statement needed to create it
func (s sqliteDialect) CreateSQL(tableName string) string {
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
func (s sqliteDialect) SelectSQL(tableName string) string {
	return fmt.Sprintf(`
		SELECT id, checksum, execution_time_in_millis, applied_at
		FROM %s
		ORDER BY id ASC
	`, tableName)
}

// QuotedTableName returns the string value of the name of the migration
// tracking table after it has been quoted for Postgres
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
