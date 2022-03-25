package schema

import (
	"database/sql"
)

// Interface verification that *sql.DB satisfies our Connection interface
var (
	_ Connection = &sql.DB{}
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

const (
	PostgresDriverName = "postgres"
	SQLiteDriverName   = "sqlite3"
	MySQLDriverName    = "mysql"
	MSSQLDriverName    = "sqlserver"
)

// TestDBs holds all of the specific database instances against which tests
// will run. The connectDB test helper references the keys of this map, and
// the withEachDB helper runs tests against every database defined here.
var TestDBs map[string]*TestDB = map[string]*TestDB{
	"postgres:latest": {
		Dialect:    Postgres,
		Driver:     PostgresDriverName,
		DockerRepo: "postgres",
		DockerTag:  "latest",
	},
	"sqlite": {
		Dialect: SQLite,
		Driver:  SQLiteDriverName,
	},
	"mysql:latest": {
		Dialect:    MySQL,
		Driver:     MySQLDriverName,
		DockerRepo: "mysql/mysql-server",
		DockerTag:  "latest",
	},
	"mariadb:latest": {
		Dialect:    MySQL,
		Driver:     MySQLDriverName,
		DockerRepo: "mariadb",
		DockerTag:  "latest",
	},
	"mssql:latest": {
		Dialect:      MSSQL,
		Driver:       MSSQLDriverName,
		DockerRepo:   "mcr.microsoft.com/mssql/server",
		DockerTag:    "2019-latest",
		SkippedArchs: []string{"arm64"},
	},
}
