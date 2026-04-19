package schema

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"runtime"
	"testing"

	containertypes "github.com/moby/moby/api/types/container"
	dockertest "github.com/ory/dockertest/v4"
)

// TestDB represents a specific database instance against which we would like
// to run database migration tests.
type TestDB struct {
	Dialect      Dialect
	Driver       string
	DockerRepo   string
	DockerTag    string
	Resource     dockertest.ClosableResource
	SkippedArchs []string
	path         string
}

func (c *TestDB) Username() string {
	switch c.Driver {
	case MSSQLDriverName:
		return "SA"
	default:
		return "schemauser"
	}
}

func (c *TestDB) Password() string {
	switch c.Driver {
	case MSSQLDriverName:
		return "Th1sI5AMor3_Compl1c4tedPasswd!"
	default:
		return "schemasecret"
	}
}

func (c *TestDB) DatabaseName() string {
	switch c.Driver {
	case MSSQLDriverName:
		return "master"
	default:
		return "schematests"
	}
}

// Port asks Docker for the host-side port we can use to connect to the
// relevant container's database port.
func (c *TestDB) Port() string {
	if c.Resource == nil {
		return ""
	}

	switch c.Driver {
	case MySQLDriverName:
		return c.Resource.GetPort("3306/tcp")
	case PostgresDriverName:
		return c.Resource.GetPort("5432/tcp")
	case MSSQLDriverName:
		return c.Resource.GetPort("1433/tcp")
	}
	return ""
}

func (c *TestDB) IsDocker() bool {
	return c.DockerRepo != "" && c.DockerTag != ""
}

func (c *TestDB) IsSQLite() bool {
	return c.Driver == SQLiteDriverName
}

// DockerEnvars computes the environment variables that are needed for a
// docker instance.
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
	case MSSQLDriverName:
		return []string{
			"ACCEPT_EULA=Y",
			fmt.Sprintf("SA_USER=%s", c.Username()),
			fmt.Sprintf("SA_PASSWORD=%s", c.Password()),
			fmt.Sprintf("SA_DATABASE=%s", c.DatabaseName()),
		}
	default:
		return []string{}
	}
}

func (c *TestDB) IsRunnable() bool {
	for _, skippedArch := range c.SkippedArchs {
		if skippedArch == runtime.GOARCH {
			return false
		}
	}
	return true
}

// Path computes the full path to the database on disk (applies only to SQLite
// instances).
func (c *TestDB) Path() string {
	switch c.Driver {
	case SQLiteDriverName:
		if c.path == "" {
			tmpF, _ := os.CreateTemp("", "schema.*.sqlite3")
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
		/**
		 * Since we want the system to be compatible with both parseTime=true and
		 * not, we use different querystrings with MariaDB and MySQL.
		 */
		if c.DockerRepo == "mariadb" {
			return fmt.Sprintf("%s:%s@(localhost:%s)/%s?parseTime=true&multiStatements=true", c.Username(), c.Password(), c.Port(), c.DatabaseName())
		}
		return fmt.Sprintf("%s:%s@(localhost:%s)/%s?multiStatements=true", c.Username(), c.Password(), c.Port(), c.DatabaseName())
	case MSSQLDriverName:
		return fmt.Sprintf("sqlserver://%s:%s@localhost:%s/?database=%s&encrypt=disable", c.Username(), c.Password(), c.Port(), c.DatabaseName())
	}
	// TODO Return error
	return "NoDSN"
}

// Init sets up a test database instance for connections. For dockertest-based
// instances, this function triggers the `docker run` call. For SQLite-based
// test instances, this creates the data file. In all cases, we verify that
// the database is connectable via a test connection.
func (c *TestDB) Init(ctx context.Context, pool dockertest.Pool) {
	var err error

	if c.IsDocker() {
		// For Docker-based test databases, we send a startup signal to have Docker
		// launch a container for this test run.
		log.Printf("Starting docker container %s:%s\n", c.DockerRepo, c.DockerTag)

		// The container is started with AutoRemove: true, and a restart policy to
		// not restart.
		c.Resource, err = pool.Run(ctx,
			c.DockerRepo,
			dockertest.WithTag(c.DockerTag),
			dockertest.WithEnv(c.DockerEnvars()),
			dockertest.WithoutReuse(),
			dockertest.WithHostConfig(func(config *containertypes.HostConfig) {
				config.RestartPolicy = containertypes.RestartPolicy{Name: "no"}
			}),
		)
		if err != nil {
			log.Fatalf("Could not start container %s:%s: %s", c.DockerRepo, c.DockerTag, err)
		}
	}

	err = pool.Retry(ctx, 0, func() error {
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
	}
	log.Printf("Successfully connected to %s", c.DSN())
}

// Connect creates an additional *database/sql.DB connection for a particular
// test database.
func (c *TestDB) Connect(t *testing.T) *sql.DB {
	db, err := sql.Open(c.Driver, c.DSN())
	if err != nil {
		t.Error(err)
	}
	return db
}

// Cleanup should be called after all tests with a database instance are
// complete. For dockertest-based tests, it deletes the docker containers.
// For SQLite tests, it deletes the database file from the temp directory.
func (c *TestDB) Cleanup(ctx context.Context) {
	var err error

	switch {
	case c.Driver == SQLiteDriverName:
		err = os.Remove(c.Path())
		if os.IsNotExist(err) {
			// Ignore error cleaning up nonexistent file
			err = nil
		}

	case c.IsDocker() && c.Resource != nil:
		err = c.Resource.Close(ctx)
	}

	if err != nil {
		log.Fatalf("Could not cleanup %s: %s", c.DSN(), err)
	}
}
