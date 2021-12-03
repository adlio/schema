package schema

import (
	"fmt"
	"strings"
	"testing"
)

func TestGetAppliedMigrations(t *testing.T) {
	withEachTestDB(t, func(t *testing.T, tdb *TestDB) {
		db := tdb.Connect(t)
		defer func() { _ = db.Close() }()

		migrator := makeTestMigrator(WithDialect(tdb.Dialect))
		migrations := makeValidButUselessMigrations()
		err := migrator.Apply(db, migrations)
		if err != nil {
			t.Error(err)
		}

		expectedCount := len(migrations)
		applied, err := migrator.GetAppliedMigrations(db)
		if err != nil {
			t.Error(err)
		}
		if len(applied) != expectedCount {
			t.Errorf("Expected %d applied migrations. Got %d", expectedCount, len(applied))
		}
	})
}

func TestGetAppliedMigrationsErrorsWhenTheTableDoesntExist(t *testing.T) {
	withEachTestDB(t, func(t *testing.T, tdb *TestDB) {
		db := tdb.Connect(t)
		defer func() { _ = db.Close() }()

		migrator := makeTestMigrator()
		migrations, err := migrator.GetAppliedMigrations(db)
		if err == nil {
			t.Error("Expected an error. Got none.")
		}
		if len(migrations) > 0 {
			t.Error("Expected empty list of applied migrations")
		}

	})
}

func TestGetAppliedMigrationsHasFriendlyScanError(t *testing.T) {
	// We are only testing PostgreSQL here because making the row scan
	// fail requires the structure of the table to change, and ALTER TABLE
	// DDL is inconsistent across database vendors.
	withTestDB(t, "postgres:latest", func(t *testing.T, tdb *TestDB) {
		db := tdb.Connect(t)
		defer func() { _ = db.Close() }()

		// First we apply a handful of migrations
		migrator := makeTestMigrator(WithDialect(tdb.Dialect))
		migrations := makeValidButUselessMigrations()
		err := migrator.Apply(db, migrations)
		if err != nil {
			t.Error(err)
		}

		// Then we re-type columns in schema_migrations, which will break
		// the rows.Scan(). This simulates a scenario where a rogue DBA has
		// messed with the schema_migrations table.
		sql := `ALTER TABLE %s ALTER COLUMN execution_time_in_millis TYPE VARCHAR`
		_, err = db.Exec(fmt.Sprintf(sql, migrator.QuotedTableName()))
		if err != nil {
			t.Error(err)
		}
		sql = `UPDATE %s SET execution_time_in_millis = 'NOT A NUMBER'`
		_, err = db.Exec(fmt.Sprintf(sql, migrator.QuotedTableName()))
		if err != nil {
			t.Error(err)
		}

		// Now we get the AppliedMigration records, and we expect the query to
		// succeed, but the Scan() to fail.
		_, err = migrator.GetAppliedMigrations(db)
		if err == nil || !strings.Contains(err.Error(), migrator.TableName) {
			t.Errorf("Expected an error referencing the schema_migrations table's name, got %s", err)
		}
	})
}
