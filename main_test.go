package schema

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/ory/dockertest"
)

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

	// Disable logging for MySQL while we await startup of the Docker containero
	// This avoids "[mysql] unexpected EOF" logging input during the delay
	// while the docker containers launch
	_ = mysql.SetLogger(nullMySQLLogger{})

	var wg sync.WaitGroup
	for name := range TestDBs {
		testDB := TestDBs[name]
		wg.Add(1)
		go func() {
			if testDB.IsRunnable() {
				testDB.Init(pool)
			}
			wg.Done()
		}()
	}
	wg.Wait()

	// Restore the default MySQL logger after we successfully connect
	// So that MySQL Driver errors appear as expected
	_ = mysql.SetLogger(log.New(os.Stderr, "[mysql] ", log.Ldate|log.Ltime|log.Lshortfile))

	code := m.Run()

	// Purge all the containers we created
	// You can't defer this because os.Exit doesn't execute defers
	for _, info := range TestDBs {
		info.Cleanup(pool)
	}

	os.Exit(code)
}

func withEachDialect(t *testing.T, f func(t *testing.T, d Dialect)) {
	dialects := []Dialect{Postgres, MySQL, SQLite, MSSQL}
	for _, dialect := range dialects {
		t.Run(fmt.Sprintf("%T", dialect), func(t *testing.T) {
			f(t, dialect)
		})
	}
}

func withEachTestDB(t *testing.T, f func(t *testing.T, tdb *TestDB)) {
	for dbName, tdb := range TestDBs {
		t.Run(dbName, func(t *testing.T) {
			if tdb.IsRunnable() {
				f(t, tdb)
			} else {
				t.Skipf("Not runnable on %s", runtime.GOARCH)
			}
		})
	}
}

func withTestDB(t *testing.T, name string, f func(t *testing.T, tdb *TestDB)) {
	tdb, exists := TestDBs[name]
	if !exists {
		t.Fatalf("Database '%s' doesn't exist. Add it to TestDBs", name)
	}
	f(t, tdb)
}

func connectDB(t *testing.T, name string) *sql.DB {
	info, exists := TestDBs[name]
	if !exists {
		t.Fatalf("Database '%s' doesn't exist.", name)
	}
	db, err := sql.Open(info.Driver, info.DSN())
	if err != nil {
		t.Fatalf("Failed to connect to %s: %s", name, err)
	}
	return db
}
