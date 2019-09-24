package schema

import "time"

// Migration is a yet-to-be-run change to the schema
type Migration struct {
	ID     string
	Script string
}

// AppliedMigration is a schema change which was successfully
// completed
type AppliedMigration struct {
	Migration
	Checksum              string
	ExecutionTimeInMillis int
	AppliedAt             time.Time
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
