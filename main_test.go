package schema

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

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
	"postgres11": {
		Driver:     "postgres",
		DockerRepo: "postgres",
		DockerTag:  "11",
	},
	"sqlite": {
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

	// Purge all the containers we created
	// You can't defer this because os.Exit doesn't execute defers
	for connType, info := range DBConns {
		if info.Resource != nil {
			if err := pool.Purge(info.Resource); err != nil {
				log.Fatalf("Could not purge	resource: %s", err)
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

func withEachDB(t *testing.T, f func(db *sql.DB)) {
	for dbName := range DBConns {
		t.Run(dbName, func(t *testing.T) {
			db := connectDB(t, dbName)
			f(db)
		})
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
