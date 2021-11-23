package schema

import "time"

// AppliedMigration represents a successfully-executed migration. It embeds
// Migration, and adds fields for execution results. This type is what
// records persisted in the schema_migrations table align with.
type AppliedMigration struct {
	Migration

	// Checksum is the MD5 hash of the Script for this migration
	Checksum string

	// ExecutionTimeInMillis is populated after the migration is run, indicating
	// how much time elapsed while the Script was executing.
	ExecutionTimeInMillis int

	// AppliedAt is the time at which this particular migration's Script began
	// executing (not when it completed executing).
	AppliedAt time.Time
}

// GetAppliedMigrations retrieves all already-applied migrations in a map keyed
// by the migration IDs
//
func (m Migrator) GetAppliedMigrations(db Queryer) (applied map[string]*AppliedMigration, err error) {
	applied = make(map[string]*AppliedMigration)
	migrations := make([]*AppliedMigration, 0)

	rows, err := db.Query(m.Dialect.SelectSQL(m.QuotedTableName()))
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		migration := AppliedMigration{}
		err = rows.Scan(&migration.ID, &migration.Checksum, &migration.ExecutionTimeInMillis, &migration.AppliedAt)
		migrations = append(migrations, &migration)
	}
	for _, migration := range migrations {
		applied[migration.ID] = migration
	}
	return applied, err
}
