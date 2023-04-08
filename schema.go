package schema

import (
	"context"
	"database/sql"
	"errors"
)

// DefaultTableName defines the name of the database table which will
// hold the status of applied migrations
const DefaultTableName = "schema_migrations"

// ErrNilDB is thrown when the database pointer is nil
var ErrNilDB = errors.New("DB pointer is nil")

// DB  defines the interface for a *sql.DB, which can be used to get a concrete
// connection to the database.
type DB interface {
	Conn(ctx context.Context) (*sql.Conn, error)
}

// Connection defines the interface for a *sql.Conn, which can both start a new
// transaction and run queries.
type Connection interface {
	Transactor
	Queryer
}

// Queryer is something which can execute a Query (either a sql.DB
// or a sql.Tx)
type Queryer interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

// Transactor defines the interface for the Begin method from the *sql.DB
type Transactor interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}
