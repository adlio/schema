package schema

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	// Postgres database driver
	_ "github.com/lib/pq"

	"github.com/ory/dockertest"
)

var postgres11DB *sql.DB

// TestMain replaces the normal test runner for this package. It connects to
// Docker running on the local machine and launches testing database
// containers to which we then connect and store the connection in a package
// global variable
//
func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Can't run schema tests. Docker is not running: %s", err)
	}

	resource, err := pool.Run("postgres", "11", []string{
		"POSTGRES_USER=postgres",
		"POSTGRES_PASSWORD=secret",
		"POSTGRES_DB=schematests",
	})
	if err != nil {
		log.Fatalf("Could not start container: %s", err)
	}

	// Prevents containers from accumulating due to failed test runs
	resource.Expire(60)

	if err = pool.Retry(func() error {
		var err error
		postgres11DB, err = sql.Open(
			"postgres",
			fmt.Sprintf("postgres://postgres:secret@localhost:%s/schematests?sslmode=disable", resource.GetPort("5432/tcp")),
		)
		if err != nil {
			return err
		}
		return postgres11DB.Ping()
	}); err != nil {
		log.Fatalf("Could not connect to container: %s", err)
		return
	}

	code := m.Run()

	// You can't defer this because os.Exit doesn't execute defers
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge	resource: %s", err)
	}

	os.Exit(code)
}

func TestGetAppliedMigrationsErrorsWhenNoneExist(t *testing.T) {
	migrator := NewMigrator(WithTableName(time.Now().Format(time.RFC3339Nano)))
	migrations, err := migrator.GetAppliedMigrations(postgres11DB)
	if err == nil {
		t.Error("Expected an error. Got none.")
	}
	if len(migrations) > 0 {
		t.Error("Expected empty list of applied migrations")
	}
}

func TestApplyWithNilDBProvidesHelpfulError(t *testing.T) {
	err := NewMigrator().Apply(nil, []*Migration{
		{
			ID:     "2019-01-01 Test",
			Script: "CREATE TABLE fake_table (id INTEGER)",
		},
	})
	if !errors.Is(err, ErrNilDB) {
		t.Errorf("Expected %v, got %v", ErrNilDB, err)
	}
}

func TestFailedMigration(t *testing.T) {
	tableName := time.Now().Format(time.RFC3339Nano)
	migrator := NewMigrator(WithTableName(tableName))
	migrations := []*Migration{
		{
			ID:     "2019-01-01 Bad Migration",
			Script: "CREATE TIBBLE bad_table_name (id INTEGER NOT NULL PRIMARY KEY)",
		},
	}
	err := migrator.Apply(postgres11DB, migrations)
	if err == nil || !strings.Contains(err.Error(), "TIBBLE") {
		t.Errorf("Expected explanatory error from failed migration. Got %v", err)
	}
	rows, err := postgres11DB.Query("SELECT * FROM " + migrator.QuotedTableName())
	defer rows.Close()
	if err != nil {
		t.Error(err)
	}
	if rows.Next() {
		t.Error("Record was inserted in tracking table even though the migration failed")
	}
}

func TestMigrationsAppliedLexicalOrderByID(t *testing.T) {
	tableName := time.Now().Format(time.RFC3339Nano)
	migrator := NewMigrator(WithDialect(Postgres), WithTableName(tableName))
	outOfOrderMigrations := []*Migration{
		{
			ID:     "2019-01-01 999 Should Run Last",
			Script: "CREATE TABLE last_table (id INTEGER NOT NULL);",
		},
		{
			ID:     "2019-01-01 001 Should Run First",
			Script: "CREATE TABLE first_table (id INTEGER NOT NULL);",
		},
	}
	err := migrator.Apply(postgres11DB, outOfOrderMigrations)
	if err != nil {
		t.Error(err)
	}

	applied, err := migrator.GetAppliedMigrations(postgres11DB)
	if err != nil {
		t.Error(err)
	}
	if len(applied) != 2 {
		t.Errorf("Expected exactly 2 applied migrations. Got %d", len(applied))
	}
	firstMigration := applied["2019-01-01 001 Should Run First"]
	if firstMigration == nil {
		t.Error("Missing first migration")
	}
	if firstMigration.Checksum == "" {
		t.Error("Expected checksum to get populated when migration ran")
	}

	secondMigration := applied["2019-01-01 999 Should Run Last"]
	if secondMigration == nil {
		t.Error("Missing second migration")
	}
	if secondMigration.Checksum == "" {
		t.Error("Expected checksum to get populated when migration ran")
	}

	if firstMigration.AppliedAt.After(secondMigration.AppliedAt) {
		t.Errorf("Expected migrations to run in lexical order, but first migration ran at %s and second one ran at %s", firstMigration.AppliedAt, secondMigration.AppliedAt)
	}
}

func TestMigrationRecoversFromPanics(t *testing.T) {
	err := transaction(postgres11DB, func(tx *sql.Tx) error {
		panic(errors.New("Panic Error"))
	})
	if err.Error() != "Panic Error" {
		t.Errorf("Expected panic to be converted to error=Panic Error. Got %v", err)
	}
	err = transaction(postgres11DB, func(tx *sql.Tx) error {
		panic("Panic String")
	})
	if err.Error() != "Panic String" {
		t.Errorf("Expected panic to be converted to error=Panic String. Got %v", err)
	}
}
