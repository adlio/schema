// Package schema provides tools to manage database schema changes
// ("migrations") as embedded functionality inside another application
// which is using a database/sql
//
// Basic usage instructions involve creating a schema.Migrator via the
// schema.NewMigrator() function, and then passing your *sql.DB
// to its .Apply() method.
//
// See the package's README.md file for more usage instructions.
package schema
