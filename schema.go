package schema

import (
	"database/sql"
	"errors"
	"fmt"
)

// DefaultTableName defines the name of the database table which will
// hold the status of applied migrations
const DefaultTableName = "schema_migrations"

// ErrNilDB is thrown when the database pointer is nil
var ErrNilDB = errors.New("DB pointer is nil")

// Connection defines the interface for a *sql.DB, which can both start a new
// transaction and run queries.
//
type Connection interface {
	Transactor
	Queryer
}

// Queryer is something which can execute a Query (either a sql.DB
// or a sql.Tx)
type Queryer interface {
	Exec(sql string, args ...interface{}) (sql.Result, error)
	Query(sql string, args ...interface{}) (*sql.Rows, error)
}

// Transactor defines the interface for the Begin method from the *sql.DB
//
type Transactor interface {
	Begin() (*sql.Tx, error)
}

// transaction wraps the supplied function in a transaction with the supplied
// database connecion
//
func transaction(db Transactor, f func(Queryer) error) (err error) {
	if db == nil {
		return ErrNilDB
	}
	tx, err := db.Begin()
	if err != nil {
		return
	}

	defer func() {
		if p := recover(); p != nil {
			switch p := p.(type) {
			case error:
				err = p
			default:
				err = fmt.Errorf("%s", p)
			}
		}
		if err != nil {
			_ = tx.Rollback()
			return
		}
		err = tx.Commit()
	}()

	return f(tx)
}
