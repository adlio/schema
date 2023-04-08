package schema

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

// TestCreateMigrationsTable ensures that each dialect and test database can
// successfully create the schema_migrations table.
func TestCreateMigrationsTable(t *testing.T) {
	withEachTestDB(t, func(t *testing.T, tdb *TestDB) {

		db := tdb.Connect(t)
		defer func() { _ = db.Close() }()

		migrator := makeTestMigrator(WithDialect(tdb.Dialect))
		err := tdb.Dialect.CreateMigrationsTable(migrator.ctx, db, migrator.QuotedTableName())
		if err != nil {
			t.Errorf("Error occurred when creating migrations table: %s", err)
		}

		// Test that we can re-run it again with no error
		err = tdb.Dialect.CreateMigrationsTable(migrator.ctx, db, migrator.QuotedTableName())
		if err != nil {
			t.Errorf("Calling createMigrationsTable a second time failed: %s", err)
		}
	})
}

// TestLockAndUnlock tests the Lock and Unlock mechanisms of each dialect and
// test database in isolation from any migrations actually being run.
func TestLockAndUnlock(t *testing.T) {
	withEachTestDB(t, func(t *testing.T, tdb *TestDB) {

		db := tdb.Connect(t)
		defer func() { _ = db.Close() }()

		migrator := makeTestMigrator(WithDialect(tdb.Dialect))

		if _, isLocker := tdb.Dialect.(Locker); isLocker {
			err := migrator.lock(db)
			if err != nil {
				t.Fatal(err)
			}

			err = migrator.unlock(db)
			if err != nil {
				t.Fatal(err)
			}
		}
	})
}

// TestApplyInLexicalOrder ensures that each dialect runs migrations in their
// lexical order rather than the order they were provided in the slice. This is
// also the primary test to assert that the data in the tracking table is
// all correct.
func TestApplyInLexicalOrder(t *testing.T) {
	withEachTestDB(t, func(t *testing.T, tdb *TestDB) {

		db := tdb.Connect(t)
		defer func() { _ = db.Close() }()

		start := time.Now().Truncate(time.Second) // MySQL has only second accuracy, so we need start/end to span 1 second

		tableName := "lexical_order_migrations"
		migrator := NewMigrator(WithDialect(tdb.Dialect), WithTableName(tableName))
		err := migrator.Apply(db, unorderedMigrations())
		if err != nil {
			t.Error(err)
		}

		end := time.Now().Add(time.Second).Truncate(time.Second) // MySQL has only second accuracy, so we need start/end to span 1 second

		applied, err := migrator.GetAppliedMigrations(db)
		if err != nil {
			t.Error(err)
		}
		if len(applied) != 3 {
			t.Errorf("Expected exactly 2 applied migrations. Got %d", len(applied))
		}

		firstMigration := applied["2021-01-01 001"]
		if firstMigration == nil {
			t.Fatal("Missing first migration")
		}
		if firstMigration.Checksum == "" {
			t.Error("Expected non-blank Checksum value after successful migration")
		}
		if firstMigration.ExecutionTimeInMillis < 1 {
			t.Errorf("Expected ExecutionTimeInMillis of %s to be tracked. Got %d", firstMigration.ID, firstMigration.ExecutionTimeInMillis)
		}
		// Put value in consistent timezone to aid error message readability
		appliedAt := firstMigration.AppliedAt.Round(time.Second)
		if appliedAt.IsZero() || appliedAt.Before(start) || appliedAt.After(end) {
			t.Errorf("Expected AppliedAt between %s and %s, got %s", start, end, appliedAt)
		}
		assertZonesMatch(t, start, appliedAt)

		secondMigration := applied["2021-01-01 002"]
		if secondMigration == nil {
			t.Fatal("Missing second migration")
		} else if secondMigration.Checksum == "" {
			t.Fatal("Expected checksum to get populated when migration ran")
		}

		if firstMigration.AppliedAt.After(secondMigration.AppliedAt) {
			t.Errorf("Expected migrations to run in lexical order, but first migration ran at %s and second one ran at %s", firstMigration.AppliedAt, secondMigration.AppliedAt)
		}
	})
}

// TestFailedMigration ensures that a migration with a syntax error triggers
// an expected error when Apply() is run. This test is run on every dialect
// and every test database instance
func TestFailedMigration(t *testing.T) {
	withEachTestDB(t, func(t *testing.T, tdb *TestDB) {

		db := tdb.Connect(t)
		defer func() { _ = db.Close() }()

		tableName := time.Now().Format(time.RFC3339Nano)
		migrator := NewMigrator(WithTableName(tableName), WithDialect(tdb.Dialect))
		migrations := []*Migration{
			{
				ID:     "2019-01-01 Bad Migration",
				Script: "CREATE TIBBLE bad_table_name (id INTEGER NOT NULL PRIMARY KEY)",
			},
		}
		err := migrator.Apply(db, migrations)
		expectErrorContains(t, err, "TIBBLE")

		query := "SELECT * FROM " + migrator.QuotedTableName()
		rows, _ := db.Query(query)

		// We expect either an error (because the transaction was rolled back
		// and the table no longer exists)... or  a query with no results
		if rows != nil {
			if rows.Next() {
				t.Error("Record was inserted in tracking table even though the migration failed")
			}
			_ = rows.Close()
		}
	})

}

// TestSimultaneousApply creates multiple Migrators and multiple distinct
// connections to each test database and attempts to call .Apply() on them all
// concurrently. The migrations include an INSERT statement, which allows us
// to count to ensure that each unique migration was only run once.
func TestSimultaneousApply(t *testing.T) {
	concurrency := 4
	dataTable := fmt.Sprintf("data%d", rand.Int()) // #nosec we don't need cryptographic security here
	migrationsTable := fmt.Sprintf("Migrations %s", time.Now().Format(time.RFC3339Nano))
	sharedMigrations := []*Migration{
		{
			ID:     "2020-05-02 Create Data Table",
			Script: fmt.Sprintf(`CREATE TABLE %s (number INTEGER)`, dataTable),
		},
		{
			ID:     "2020-05-03 Add Initial Record",
			Script: fmt.Sprintf(`INSERT INTO %s (number) VALUES (1)`, dataTable),
		},
	}

	withEachTestDB(t, func(t *testing.T, tdb *TestDB) {
		var wg sync.WaitGroup
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(i int) {
				db := tdb.Connect(t)
				defer func() { _ = db.Close() }()

				migrator := NewMigrator(WithDialect(tdb.Dialect), WithTableName(migrationsTable))
				err := migrator.Apply(db, sharedMigrations)
				if err != nil {
					t.Error(err)
				}
				_, err = db.Exec(fmt.Sprintf("INSERT INTO %s (number) VALUES (1)", dataTable))
				if err != nil {
					t.Error(err)
				}
				wg.Done()
			}(i)
		}
		wg.Wait()

		// We expect concurrency + 1 rows in the data table
		// (1 from the migration, and one each for the
		// goroutines which ran Apply and then did an
		// insert afterwards)
		db := tdb.Connect(t)
		defer func() { _ = db.Close() }()

		count := 0
		row := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", dataTable))
		err := row.Scan(&count)
		if err != nil {
			t.Error(err)
		}
		if count != concurrency+1 {
			t.Errorf("Expected to get %d rows in %s table. Instead got %d", concurrency+1, dataTable, count)
		}

	})
}

// TestMultiSchemaSupport ensures that each dialect and test database support
// having multiple tracking tables each tracking separate sets of migrations.
//
// The test scenario here is one set of "music" migrations which deal with
// artists, albums and tracks, and a separate set of "contacts" migrations
// which deal with contacts, phone_numbers, and addresses.
func TestMultiSchemaSupport(t *testing.T) {
	withEachTestDB(t, func(t *testing.T, tdb *TestDB) {
		music := NewMigrator(WithDialect(tdb.Dialect), WithTableName("music_migrations"))
		contacts := NewMigrator(WithDialect(tdb.Dialect), WithTableName("contacts_migrations"))

		// Use the same connection for both sets of migrations
		db := tdb.Connect(t)
		defer func() { _ = db.Close() }()

		// Apply the Music migrations
		err := music.Apply(db, testMigrations(t, "music"))
		if err != nil {
			t.Errorf("Failed to apply music migrations: %s", err)
		}

		// ... then the Contacts Migrations
		err = contacts.Apply(db, testMigrations(t, "contacts"))
		if err != nil {
			t.Errorf("Failed to apply contact migrations: %s", err)
		}

		// Then run a SELECT COUNT(*) query on each table to ensure that all of the
		// expected tables are co-existing in the same database and that they all
		// contain the expected number of rows (this approach is admittedly odd,
		// but it relies only on ANSI SQL code, so it should run on any SQL database).
		expectedRowCounts := map[string]int{
			"music_migrations":    3,
			"contacts_migrations": 3,
			"contacts":            1,
			"phone_numbers":       3,
			"addresses":           2,
			"artists":             0,
			"albums":              0,
			"tracks":              0,
		}
		for table, expectedRowCount := range expectedRowCounts {
			qtn := tdb.Dialect.QuotedTableName("", table)
			actualCount := -1 // Don't initialize to 0 because that's an expected value
			query := fmt.Sprintf("SELECT COUNT(*) FROM %s", qtn)
			rows, err := db.Query(query)
			if err != nil {
				t.Error(err)
			}
			if rows != nil && rows.Next() {
				err = rows.Scan(&actualCount)
				if err != nil {
					t.Error(err)
				}
			} else {
				t.Errorf("Expected rows")
			}
			if actualCount != expectedRowCount {
				t.Errorf("Expected %d rows in table %s. Got %d", expectedRowCount, qtn, actualCount)
			}
		}
	})
}

// TestRunFailure ensures that a low-level connection or query-related failure
// triggers an expected error.
func TestRunFailure(t *testing.T) {
	bq := BadQueryer{}
	m := makeTestMigrator()
	err := m.run(bq, testMigrations(t, "useless-ansi"))
	expectErrorContains(t, err, "SELECT id, checksum")

	err = m.run(nil, testMigrations(t, "useless-ansi"))
	if err != ErrNilDB {
		t.Errorf("Expected error '%s'. Got '%v'.", ErrNilDB, err)
	}
}

func TestNewMigratorApplyChain(t *testing.T) {
	// This is a compilability test... it is here to confirm that
	// NewMigrator()'s return value can have Apply() called on it.
	_ = NewMigrator().Apply(nil, testMigrations(t, "useless-ansi"))
}

// makeTestMigrator is a utility function which produces a migrator with an
// isolated environment (isolated due to a unique name for the migration
// tracking table).
func makeTestMigrator(options ...Option) *Migrator {
	tableName := time.Now().Format(time.RFC3339Nano)
	options = append(options, WithTableName(tableName))
	return NewMigrator(options...)
}

func testMigrations(t *testing.T, dirName string) []*Migration {
	path := fmt.Sprintf("test-migrations/%s", dirName)
	migrations, err := MigrationsFromDirectoryPath(path)
	if err != nil {
		t.Fatalf("Failed to load test migrations from '%s'", path)
	}
	return migrations
}

// assertZonesMatch accepts two Times and fails the test if their time zones
// don't match.
func assertZonesMatch(t *testing.T, expected, actual time.Time) {
	t.Helper()
	expectedName, expectedOffset := expected.Zone()
	actualName, actualOffset := actual.Zone()
	if expectedOffset != actualOffset {
		t.Errorf("Expected Zone '%s' with offset %d. Got Zone '%s' with offset %d", expectedName, expectedOffset, actualName, actualOffset)
	}
}
