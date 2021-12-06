package schema

import (
	"fmt"
	"time"
)

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

type applyTime struct {
	Value time.Time
}

func (t *applyTime) Scan(src interface{}) (err error) {
	if src == nil {
		t.Value = time.Time{}
	}

	if srcTime, isTime := src.(time.Time); isTime {
		t.Value = srcTime.In(time.Local)
		return nil
	}

	return t.ScanString(fmt.Sprintf("%s", src))
}

func (t *applyTime) ScanString(src string) (err error) {
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

// GetAppliedMigrations retrieves all already-applied migrations in a map keyed
// by the migration IDs
//
func (m Migrator) GetAppliedMigrations(db Queryer) (applied map[string]*AppliedMigration, err error) {
	applied = make(map[string]*AppliedMigration)
	migrations := make([]*AppliedMigration, 0)

	rows, err := db.Query(m.Dialect.SelectSQL(m.QuotedTableName()))
	if err != nil {
		err = fmt.Errorf("failed to GetAppliedMigrations. Check the %s table?: %w", m.QuotedTableName(), err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		migration := AppliedMigration{}

		var appliedAt applyTime
		err = rows.Scan(&migration.ID, &migration.Checksum, &migration.ExecutionTimeInMillis, &appliedAt)
		if err != nil {
			err = fmt.Errorf("failed to GetAppliedMigrations. Did somebody change the structure of the %s table?: %w", m.QuotedTableName(), err)
			return applied, err
		}
		migration.AppliedAt = appliedAt.Value

		migrations = append(migrations, &migration)
	}
	for _, migration := range migrations {
		applied[migration.ID] = migration
	}
	return applied, err
}
