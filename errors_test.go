package schema

import (
	"database/sql"
	"fmt"
	"strings"
)

var (
	// ErrBeginFailed indicates that the Begin() method failed (couldn't start Tx)
	ErrBeginFailed = fmt.Errorf("begin failed")

	// ErrPriorFailure indicates a failure happened earlier in the Migrator Apply()
	ErrPriorFailure = fmt.Errorf("previous error")
)

// BadQueryer implements the Connection interface, but fails on every call to
// Exec or Query. The error message will include the SQL statement to help
// verify the "right" failure occurred.
type BadQueryer struct{}

func (bq BadQueryer) Exec(sql string, args ...interface{}) (sql.Result, error) {
	return nil, fmt.Errorf("FAIL: %s", strings.TrimSpace(sql))
}

func (bq BadQueryer) Query(sql string, args ...interface{}) (*sql.Rows, error) {
	return nil, fmt.Errorf("FAIL: %s", strings.TrimSpace(sql))
}

// BadTransactor implements the Transactor interface with no-ops for Exec() and
// Query(), and failures on all calls to Begin()
type BadTransactor struct{}

func (bt BadTransactor) Begin() (*sql.Tx, error) {
	return nil, ErrBeginFailed
}

// BadConnection implements the Connection interface, but fails on all calls to
// Begin(), Query() or Exec()
//
type BadConnection struct{}

func (bc BadConnection) Begin() (*sql.Tx, error) {
	return nil, ErrBeginFailed
}

func (bc BadConnection) Exec(sql string, args ...interface{}) (sql.Result, error) {
	return nil, fmt.Errorf("FAIL: %s", strings.TrimSpace(sql))
}

func (bc BadConnection) Query(sql string, args ...interface{}) (*sql.Rows, error) {
	return nil, fmt.Errorf("FAIL: %s", strings.TrimSpace(sql))
}
