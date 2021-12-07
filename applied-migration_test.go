package schema

import (
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
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
	withEachTestDB(t, func(t *testing.T, tdb *TestDB) {
		migrator := makeTestMigrator(WithDialect(tdb.Dialect))

		db, mock, err := sqlmock.New()
		if err != nil {
			t.Error(err)
		}

		// Build a rowset that is completely different than the AppliedMigration
		// struct is expecting to force a Scan error
		rows := sqlmock.NewRows([]string{"nonsense", "column", "names"}).AddRow(1, "trash", "data")
		mock.ExpectQuery("").RowsWillBeClosed().WillReturnRows(rows)

		_, err = migrator.GetAppliedMigrations(db)
		if err == nil || !strings.Contains(err.Error(), migrator.TableName) {
			t.Errorf("Expected an error referencing the schema_migrations table's name, got %s", err)
		}
	})
}
