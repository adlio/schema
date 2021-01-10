package schema

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	// Postgres database driver
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/ory/dockertest"
)

type ConnInfo struct {
	Driver     string
	DockerRepo string
	DockerTag  string
	DSN        string
	Resource   *dockertest.Resource
}

var DBConns map[string]*ConnInfo = map[string]*ConnInfo{
	"postgres11": &ConnInfo{
		Driver:     "postgres",
		DockerRepo: "postgres",
		DockerTag:  "11",
	},
	"sqlite": &ConnInfo{
		Driver: "sqlite3",
		DSN:    filepath.Join(os.TempDir(), fmt.Sprintf("sqlite_test_%d.db", time.Now().Unix())),
	},
}

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

	for _, info := range DBConns {
		switch info.Driver {
		case "postgres":
			// Provision the container
			info.Resource, err = pool.Run(info.DockerRepo, info.DockerTag, []string{
				"POSTGRES_USER=postgres",
				"POSTGRES_PASSWORD=secret",
				"POSTGRES_DB=schematests",
			})
			if err != nil {
				log.Fatalf("Could not start container %s:%s: %s", info.DockerRepo, info.DockerTag, err)
			}

			// Set the container to expire in a minute to avoid orphaned contianers
			// hanging around
			err = info.Resource.Expire(60)
			if err != nil {
				log.Fatalf("Could not set expiration time for docker test containers: %s", err)
			}

			// Save the DSN to make new connections later
			info.DSN = fmt.Sprintf("postgres://postgres:secret@localhost:%s/schematests?sslmode=disable", info.Resource.GetPort("5432/tcp"))

			// Wait for the database to come online
			if err = pool.Retry(func() error {
				testConn, err := sql.Open(info.Driver, info.DSN)
				if err != nil {
					return err
				}
				defer func() { _ = testConn.Close() }()
				return testConn.Ping()
			}); err != nil {
				log.Fatalf("Could not connect to container: %s", err)
				return
			}

		}
	}

	code := m.Run()

	// Purge all the resources we created
	// You can't defer this because os.Exit doesn't execute defers
	for connType, info := range DBConns {
		if info.Resource != nil {
			if err := pool.Purge(info.Resource); err != nil {
				log.Printf("Warning: could not purge docker resource: %s", err)
			}
		}

		switch connType {
		case "sqlite":
			if err := os.Remove(info.DSN); err != nil && !os.IsNotExist(err) {
				log.Printf("Warning: could not delete sqlite database: %s", err)
			}
		}
	}

	os.Exit(code)
}

func TestGetAppliedMigrationsErrorsWhenNoneExist(t *testing.T) {
	db := connectDB(t, "postgres11")
	migrator := NewMigrator(WithTableName(time.Now().Format(time.RFC3339Nano)))
	migrations, err := migrator.GetAppliedMigrations(db)
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
	db := connectDB(t, "postgres11")
	tableName := time.Now().Format(time.RFC3339Nano)
	migrator := NewMigrator(WithTableName(tableName))
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
	rows, err := db.Query("SELECT * FROM " + migrator.QuotedTableName())
	if err != nil {
		t.Error(err)
	}
	if rows.Next() {
		t.Error("Record was inserted in tracking table even though the migration failed")
	}
	_ = rows.Close()
}

func TestMigrationsAppliedLexicalOrderByID(t *testing.T) {
	db := connectDB(t, "postgres11")
	tableName := "lexical_order_migrations"
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
	err := migrator.Apply(db, outOfOrderMigrations)
	if err != nil {
		t.Error(err)
	}

	applied, err := migrator.GetAppliedMigrations(db)
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

func TestSimultaneousMigrations(t *testing.T) {
	concurrency := 4
	dataTable := fmt.Sprintf("data%d", rand.Int())
	migrationsTable := fmt.Sprintf("Migrations %s", time.Now().Format(time.RFC3339Nano))
	sharedMigrations := []*Migration{
		{
			ID:     "2020-05-01 Sleep",
			Script: "SELECT pg_sleep(1)",
		},
		{
			ID: "2020-05-02 Create Data Table",
			Script: fmt.Sprintf(`CREATE TABLE %s (
				id INTEGER GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
				created_at TIMESTAMP WITH TIME ZONE NOT NULL
			)`, dataTable),
		},
		{
			ID:     "2020-05-03 Add Initial Record",
			Script: fmt.Sprintf(`INSERT INTO %s (created_at) VALUES (NOW())`, dataTable),
		},
	}

	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(i int) {
			db := connectDB(t, "postgres11")
			migrator := NewMigrator(
				WithDialect(Postgres),
				WithTableName(migrationsTable),
			)
			err := migrator.Apply(db, sharedMigrations)
			if err != nil {
				t.Error(err)
			}
			_, err = db.Exec(fmt.Sprintf("INSERT INTO %s (created_at) VALUES (NOW())", dataTable))
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
	db := connectDB(t, "postgres11")
	count := 0
	row := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", dataTable))
	err := row.Scan(&count)
	if err != nil {
		t.Error(err)
	}
	if count != concurrency+1 {
		t.Errorf("Expected to get %d rows in %s table. Instead got %d", concurrency+1, dataTable, count)
	}

}

func TestMigrationRecoversFromPanics(t *testing.T) {
	db := connectDB(t, "postgres11")
	err := transaction(db, func(tx *sql.Tx) error {
		panic(errors.New("Panic Error"))
	})
	if err.Error() != "Panic Error" {
		t.Errorf("Expected panic to be converted to error=Panic Error. Got %v", err)
	}
	err = transaction(db, func(tx *sql.Tx) error {
		panic("Panic String")
	})
	if err.Error() != "Panic String" {
		t.Errorf("Expected panic to be converted to error=Panic String. Got %v", err)
	}
}

func connectDB(t *testing.T, name string) *sql.DB {
	info, exists := DBConns[name]
	if !exists {
		t.Errorf("Database '%s' doesn't exist.", name)
	}
	db, err := sql.Open(info.Driver, info.DSN)
	if err != nil {
		t.Error(err)
	}
	return db
}
