package schema

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
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
		migrator.createMigrationsTable(db)
		if migrator.err != nil {
			t.Errorf("Error occurred when creating migrations table: %s", migrator.err)
		}

		// Test that we can re-run it safely
		migrator.createMigrationsTable(db)
		if migrator.err != nil {
			t.Errorf("Calling createMigrationsTable a second time failed: %s", migrator.err)
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
		migrator.lock(db)
		if migrator.err != nil {
			t.Fatal(migrator.err)
		}

		migrator.unlock(db)
		if migrator.err != nil {
			t.Fatal(migrator.err)
		}
	})
}

// TestApplyInLexicalOrder ensures that each dialect runs migrations in their
// lexical order rather than the order they were provided in the slice.
//
func TestApplyInLexicalOrder(t *testing.T) {
	withEachTestDB(t, func(t *testing.T, tdb *TestDB) {

		db := tdb.Connect(t)
		defer func() { _ = db.Close() }()

		tableName := "lexical_order_migrations"
		migrator := NewMigrator(WithDialect(tdb.Dialect), WithTableName(tableName))
		err := migrator.Apply(db, makeValidUnorderedMigrations())
		if err != nil {
			t.Error(err)
		}

		applied, err := migrator.GetAppliedMigrations(db)
		if err != nil {
			t.Error(err)
		}
		if len(applied) != 3 {
			t.Errorf("Expected exactly 2 applied migrations. Got %d", len(applied))
		}
		firstMigration := applied["2021-01-01 001"]
		if firstMigration == nil {
			t.Error("Missing first migration")
		} else if firstMigration.Checksum == "" {
			t.Error("Expected checksum to get populated when migration ran")
		}

		secondMigration := applied["2021-01-01 002"]
		if secondMigration == nil {
			t.Error("Missing second migration")
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
//
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
		if err == nil || !strings.Contains(err.Error(), "TIBBLE") {
			t.Errorf("Expected explanatory error from failed migration. Got %v", err)
		}
		rows, _ := db.Query("SELECT * FROM " + migrator.QuotedTableName())

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
//
func TestSimultaneousApply(t *testing.T) {
	concurrency := 4
	dataTable := fmt.Sprintf("data%d", rand.Int())
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
		err := music.Apply(db, makeMusicMigrations())
		if err != nil {
			t.Errorf("Failed to apply music migrations: %s", err)
		}

		// ... then the Contacts Migrations
		err = contacts.Apply(db, makeContactsMigrations())
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
			sql := fmt.Sprintf("SELECT COUNT(*) FROM %s", qtn)
			rows, err := db.Query(sql)
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
//
func TestRunFailure(t *testing.T) {
	bc := BadConnection{}
	m := makeTestMigrator()
	m.run(bc, makeValidUnorderedMigrations())
	expectedContents := "FAIL: SELECT id, checksum"
	if m.err == nil || !strings.Contains(m.err.Error(), expectedContents) {
		t.Errorf("Expected error msg with '%s'. Got '%v'.", expectedContents, m.err)
	}

	m.err = ErrPriorFailure
	m.run(bc, makeValidUnorderedMigrations())
	if m.err != ErrPriorFailure {
		t.Errorf("Expected error %v. Got %v.", ErrPriorFailure, m.err)
	}

	m.err = nil
	m.run(nil, makeValidUnorderedMigrations())
	if m.err != ErrNilDB {
		t.Errorf("Expected error '%s'. Got '%v'.", expectedContents, m.err)
	}
}

func TestMigrationRecoversFromPanics(t *testing.T) {
	db := connectDB(t, "postgres11")
	migrator := makeTestMigrator()
	migrator.transaction(db, func(tx Queryer) { panic(errors.New("Panic Error")) })
	if migrator.err == nil {
		t.Error("Expected error to be set after panic. Got nil")
	} else if migrator.err.Error() != "Panic Error" {
		t.Errorf("Expected panic to be converted to error=Panic Error. Got %v", migrator.err)
	}

	migrator.err = nil
	migrator.transaction(db, func(tx Queryer) { panic("Panic String") })

	if migrator.err == nil {
		t.Error("Expected error to be set after panic. Got nil")
	} else if migrator.err.Error() != "Panic String" {
		t.Errorf("Expected panic to be converted to error=Panic String. Got %v", migrator.err)
	}
}

// makeTestMigrator is a utility function which produces a migrator with an
// isolated environment (isolated due to a unique name for the migration
// tracking table).
func makeTestMigrator(options ...Option) Migrator {
	tableName := time.Now().Format(time.RFC3339Nano)
	options = append(options, WithTableName(tableName))
	return NewMigrator(options...)
}

func makeValidUnorderedMigrations() []*Migration {
	return []*Migration{
		{
			ID: "2021-01-01 002",
			Script: `CREATE TABLE data_table (
				id INTEGER PRIMARY KEY,
				name VARCHAR(255),
				created_at TIMESTAMP
			)`,
		},
		{
			ID:     "2021-01-01 001",
			Script: "CREATE TABLE first_table (first_name VARCHAR(255), last_name VARCHAR(255))",
		},
		{
			ID:     "2021-01-01 003",
			Script: `INSERT INTO first_table (first_name, last_name) VALUES ('John', 'Doe')`,
		},
	}
}

// makeValidButUselessMigrations exists to provide some migrations that can be run
// many times in the same database without conflicts (something which would be
// useless in the real world, but which is very useful for tests).
//
// These migrations are used by the AppliedMigration tests so that we can see
// the effects on the schema_migrations table.
//
func makeValidButUselessMigrations() []*Migration {
	return []*Migration{
		{
			ID:     "0000-00-00 001",
			Script: "SELECT 1",
		},
		{
			ID:     "0000-00-00 002",
			Script: "SELECT 2",
		},
	}
}

// makeMusicMigrations generates a set of ANSI-SQL compliant migrations that
// create music-related database tables on any SQL database.
//
func makeMusicMigrations() []*Migration {
	return []*Migration{
		{
			ID:     "0000-00-00 001 Artists",
			Script: `CREATE TABLE artists (id INTEGER PRIMARY KEY)`,
		},
		{
			ID: "0000-00-00 002 Albums",
			Script: `CREATE TABLE albums (
				id INTEGER PRIMARY KEY,
				artist_id INTEGER
			)`,
		},
		{
			ID: "0000-00-00 003 Tracks",
			Script: `CREATE TABLE tracks (
				id INTEGER PRIMARY KEY,
				artist_id INTEGER,
				album_id INTEGER
			)`,
		},
	}
}

// makeContactsMigrations generates a set of ANSI-SQL compliant migrations that
// create contacts-related database tables on any SQL database. Each of these
// migrations is a multi-statement string, which requires some special
// configuration in some Go database/sql drivers (go-mysql-driver in particular).
//
func makeContactsMigrations() []*Migration {
	return []*Migration{
		{
			ID: "0000-00-00 001 Contacts",
			Script: `
				CREATE TABLE contacts (id INTEGER PRIMARY KEY);
				INSERT INTO contacts (id) VALUES (1);
			`,
		},
		{
			ID: "0000-00-00 002 Phone Numbers",
			Script: `CREATE TABLE phone_numbers (
				id INTEGER PRIMARY KEY,
				contact_id INTEGER
			);
			INSERT INTO phone_numbers (id, contact_id) VALUES (1, 1);
			INSERT INTO phone_numbers (id, contact_id) VALUES (2, 1);
			INSERT INTO phone_numbers (id, contact_id) VALUES (3, 1);`,
		},
		{
			ID: "0000-00-00 003 Addresses",
			Script: `CREATE TABLE addresses (
				id INTEGER PRIMARY KEY,
				contact_id INTEGER
			);
			INSERT INTO addresses (id, contact_id) VALUES (1,1);
			INSERT INTO addresses (id, contact_id) VALUES (2, 1);
			`,
		},
	}
}
