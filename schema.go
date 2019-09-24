package schema

import (
	"database/sql"
	"errors"
	"fmt"
	"sync"
)

// A global mutex to prevent simultaneous
// migrations
var mutex = &sync.Mutex{}

// DefaultTableName defines the name of the database table which will
// hold the status of applied migrations
const DefaultTableName = "schema_migrations"

// ErrNilDB is thrown when the database pointer is nil
var ErrNilDB = errors.New("DB pointer is nil")

// Queryer is something which can execute a Query (either a sql.DB
// or a sql.Tx))
type Queryer interface {
	Query(sql string, args ...interface{}) (*sql.Rows, error)
}

// transaction wraps the supplied function in a transaction with the supplied
// database connecion
//
func transaction(db *sql.DB, f func(*sql.Tx) error) (err error) {
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
			tx.Rollback()
			return
		}
		err = tx.Commit()
	}()

	return f(tx)
}
