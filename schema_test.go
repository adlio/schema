package schema

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"testing"

	// Postgres database driver
	_ "github.com/lib/pq"

	"github.com/ory/dockertest"
)

// Interface verification that *sql.DB satisfies our Transactor interface
var (
	_ Transactor = &sql.DB{}
)

// Interface verification that *sql.Tx and *sql.DB both satisfy our
// Queryer interface
var (
	_ Queryer = &sql.DB{}
	_ Queryer = &sql.Tx{}
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
	for _, info := range DBConns {
		if err := pool.Purge(info.Resource); err != nil {
			log.Fatalf("Could not purge	resource: %s", err)
		}
	}

	os.Exit(code)
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
