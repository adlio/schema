package schema

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
)

const (
	PostgresDriverName = "postgres"
	SQLiteDriverName   = "sqlite3"
	MySQLDriverName    = "mysql"
)

// TestDBs holds all of the specific database instances against which tests
// will run. The connectDB test helper refere ces the keys of this map, and
// the withEachDB helper runs tests against every database defined here.
var TestDBs map[string]*TestDB = map[string]*TestDB{
	"postgres:latest": {
		Dialect:    Postgres,
		Driver:     PostgresDriverName,
		DockerRepo: "postgres",
		DockerTag:  "latest",
	},
	"sqlite": {
		Dialect: NewSQLite(),
		Driver:  SQLiteDriverName,
	},
	"mysql:latest": {
		Dialect:    MySQL,
		Driver:     MySQLDriverName,
		DockerRepo: "mysql",
		DockerTag:  "latest",
	},
}

// TestDB represents a specific database instance against which we would like
// to run database migration tests.
//
type TestDB struct {
	Dialect    Dialect
	Driver     string
	DockerRepo string
	DockerTag  string
	Resource   *dockertest.Resource
	path       string
}

func (c *TestDB) Username() string {
	return "schemauser"
}

func (c *TestDB) Password() string {
	return "schemasecret"
}

func (c *TestDB) DatabaseName() string {
	return "schematests"
}

func (c *TestDB) Port() string {
	switch c.Driver {
	case MySQLDriverName:
		return c.Resource.GetPort("3306/tcp")
	case PostgresDriverName:
		return c.Resource.GetPort("5432/tcp")
	}
	return ""
}

func (c *TestDB) IsDocker() bool {
	return c.DockerRepo != "" && c.DockerTag != ""
}

func (c *TestDB) IsSQLite() bool {
	return c.Driver == SQLiteDriverName
}

func (c *TestDB) DockerEnvars() []string {
	switch c.Driver {
	case PostgresDriverName:
		return []string{
			fmt.Sprintf("POSTGRES_USER=%s", c.Username()),
			fmt.Sprintf("POSTGRES_PASSWORD=%s", c.Password()),
			fmt.Sprintf("POSTGRES_DB=%s", c.DatabaseName()),
		}
	case MySQLDriverName:
		return []string{
			"MYSQL_RANDOM_ROOT_PASSWORD=true",
			fmt.Sprintf("MYSQL_USER=%s", c.Username()),
			fmt.Sprintf("MYSQL_PASSWORD=%s", c.Password()),
			fmt.Sprintf("MYSQL_DATABASE=%s", c.DatabaseName()),
		}
	default:
		return []string{}
	}
}

func (c *TestDB) Path() string {
	switch c.Driver {
	case SQLiteDriverName:
		if c.path == "" {
			tmpF, _ := ioutil.TempFile("", "schema.*.sqlite3")
			c.path = tmpF.Name()
		}
		return c.path
	default:
		return ""
	}
}

func (c *TestDB) DSN() string {
	switch c.Driver {
	case PostgresDriverName:
		return fmt.Sprintf("postgres://%s:%s@localhost:%s/%s?sslmode=disable", c.Username(), c.Password(), c.Port(), c.DatabaseName())
	case SQLiteDriverName:
		return c.Path()
	case MySQLDriverName:
		return fmt.Sprintf("%s:%s@(localhost:%s)/%s?parseTime=true&multiStatements=true", c.Username(), c.Password(), c.Port(), c.DatabaseName())
	}
	// TODO Return error
	return "NoDSN"
}

func (c *TestDB) Init(pool *dockertest.Pool) {
	var err error

	if c.IsDocker() {
		// For Docker-based test databases, we send a startup signal to have Docker
		// launch a container for this test run.
		log.Printf("Starting docker container %s:%s\n", c.DockerRepo, c.DockerTag)

		if c.Driver == MySQLDriverName {
			// Disable logging for MySQL while we await startup of the Docker container
			mysql.SetLogger(nullMySQLLogger{})
		}

		// The container is started with AutoRemove: true, and a restart policy to
		// not restart
		c.Resource, err = pool.RunWithOptions(&dockertest.RunOptions{
			Repository: c.DockerRepo,
			Tag:        c.DockerTag,
			Env:        c.DockerEnvars(),
		}, func(config *docker.HostConfig) {
			config.AutoRemove = true
			config.RestartPolicy = docker.RestartPolicy{
				Name: "no",
			}
		})

		if err != nil {
			log.Fatalf("Could not start container %s:%s: %s", c.DockerRepo, c.DockerTag, err)
		}

		// Even if everything goes OK, kill off the container after n seconds
		c.Resource.Expire(120)
	}

	// Regardless of whether the DB is docker-based on not, we use the pool's
	// exponential backoff helper to wait until connections succeed for this
	// database
	err = pool.Retry(func() error {
		testConn, err := sql.Open(c.Driver, c.DSN())
		if err != nil {
			return err
		}

		// We close the test connection... other code will re-open via the DSN()
		defer func() { _ = testConn.Close() }()
		return testConn.Ping()
	})
	if err != nil {
		log.Fatalf("Could not connect to %s: %s", c.DSN(), err)
	} else {
		log.Printf("Successfully connected to %s", c.DSN())
	}

	if c.Driver == MySQLDriverName {
		// Restore the default MySQL logger after we successfully connect
		mysql.SetLogger(log.New(os.Stderr, "[mysql] ", log.Ldate|log.Ltime|log.Lshortfile))
	}
}

func (c *TestDB) Connect(t *testing.T) *sql.DB {
	db, err := sql.Open(c.Driver, c.DSN())
	if err != nil {
		t.Error(err)
	}
	return db
}

func (c *TestDB) Cleanup(pool *dockertest.Pool) {
	var err error

	switch {
	case c.Driver == SQLiteDriverName:
		err = os.Remove(c.Path())
		if os.IsNotExist(err) {
			// Ignore error cleaning up nonexistent file
			err = nil
		}

	case c.IsDocker() && c.Resource != nil:
		err = pool.Purge(c.Resource)
	}

	if err != nil {
		log.Fatalf("Could not cleanup %s: %s", c.DSN(), err)
	}
}
