package schema

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

var (
	// ErrConnFailed indicates that the Conn() method failed (couldn't get a connection)
	ErrConnFailed = fmt.Errorf("connect failed")

	// ErrBeginFailed indicates that the Begin() method failed (couldn't start Tx)
	ErrBeginFailed = fmt.Errorf("begin failed")

	// ErrPriorFailure indicates a failure happened earlier in the Migrator Apply()
	ErrPriorFailure = fmt.Errorf("previous error")

	ErrLockFailed = fmt.Errorf("lock failed")
)

// BadQueryer implements the Connection interface, but fails on every call to
// Exec or Query. The error message will include the SQL statement to help
// verify the "right" failure occurred.
type BadQueryer struct{}

func (bq BadQueryer) ExecContext(ctx context.Context, sql string, args ...interface{}) (sql.Result, error) {
	return nil, fmt.Errorf("FAIL: %s", strings.TrimSpace(sql))
}

func (bq BadQueryer) QueryContext(ctx context.Context, sql string, args ...interface{}) (*sql.Rows, error) {
	return nil, fmt.Errorf("FAIL: %s", strings.TrimSpace(sql))
}

// BadTransactor implements the Transactor interface with no-ops for Exec() and
// Query(), and failures on all calls to Begin()
type BadTransactor struct{}

func (bt BadTransactor) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return nil, ErrBeginFailed
}

// BadConnection implements the Connection interface, but fails on all calls to
// Begin(), Query() or Exec()
//
type BadConnection struct{}

func (bc BadConnection) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return nil, ErrBeginFailed
}

func (bc BadConnection) ExecContext(ctx context.Context, sql string, args ...interface{}) (sql.Result, error) {
	return nil, fmt.Errorf("FAIL: %s", strings.TrimSpace(sql))
}

func (bc BadConnection) QueryContext(ctx context.Context, sql string, args ...interface{}) (*sql.Rows, error) {
	return nil, fmt.Errorf("FAIL: %s", strings.TrimSpace(sql))
}

// BadDB implements the interface for the *sql.DB Conn() method in a way that
// always fails
type BadDB struct{}

func (bd BadDB) Conn(ctx context.Context) (*sql.Conn, error) {
	return nil, ErrConnFailed
}

func TestApplyWithNilDBProvidesHelpfulError(t *testing.T) {
	withEachDialect(t, func(t *testing.T, d Dialect) {
		migrator := NewMigrator(WithDialect(d))
		err := migrator.Apply(nil, makeValidUnorderedMigrations())
		if !errors.Is(err, ErrNilDB) {
			t.Errorf("Expected %v, got %v", ErrNilDB, err)
		}
	})
}

func TestApplyWithNoMigrations(t *testing.T) {
	db, _, _ := sqlmock.New()
	migrator := NewMigrator()
	err := migrator.Apply(db, []*Migration{})
	if err != nil {
		t.Errorf("Expected no error when running no migrations, got %s", err)
	}
}
func TestApplyConnFailure(t *testing.T) {
	bd := BadDB{}
	migrator := Migrator{}
	err := migrator.Apply(bd, makeValidUnorderedMigrations())
	if err != ErrConnFailed {
		t.Errorf("Expected %v, got %v", ErrConnFailed, err)
	}
}

func TestApplyLockFailure(t *testing.T) {
	migrator := NewMigrator(WithDialect(Postgres))

	db, mock, _ := sqlmock.New()
	mock.ExpectExec("^SELECT pg_advisory_lock").WillReturnError(ErrLockFailed)
	err := migrator.Apply(db, makeValidUnorderedMigrations())
	if err != ErrLockFailed {
		t.Errorf("Expected err '%s', got '%s'", ErrLockFailed, err)
	}
}

func TestApplyBeginFailure(t *testing.T) {
	migrator := NewMigrator(WithDialect(Postgres))

	db, mock, _ := sqlmock.New()
	mock.ExpectExec("^SELECT pg_advisory_lock").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectBegin().WillReturnError(ErrBeginFailed)
	mock.ExpectExec("^SELECT pg_advisory_unlock").WillReturnResult(sqlmock.NewResult(0, 0))
	err := migrator.Apply(db, makeValidUnorderedMigrations())
	if err != ErrBeginFailed {
		t.Errorf("Expected err '%s', got '%s'", ErrBeginFailed, err)
	}
}

func TestApplyCreateFailure(t *testing.T) {
	migrator := NewMigrator(WithDialect(Postgres))

	db, mock, _ := sqlmock.New()
	mock.ExpectExec("^SELECT pg_advisory_lock").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectBegin()
	expectedErr := fmt.Errorf("CREATE TABLE statement failed")
	mock.ExpectExec("^CREATE TABLE").WillReturnError(expectedErr)
	mock.ExpectRollback()
	mock.ExpectExec("^SELECT pg_advisory_unlock").WillReturnResult(sqlmock.NewResult(0, 0))
	err := migrator.Apply(db, makeValidUnorderedMigrations())
	if err != expectedErr {
		t.Errorf("Expected err '%s', got '%s'", expectedErr, err)
	}
}

func TestLockFailure(t *testing.T) {
	bq := BadQueryer{}
	migrator := NewMigrator(WithDialect(Postgres))
	err := migrator.lock(bq)
	expectErrorContains(t, err, "SELECT pg_advisory_lock")
}

func TestUnlockFailure(t *testing.T) {
	bq := BadQueryer{}
	migrator := NewMigrator(WithDialect(Postgres))
	err := migrator.unlock(bq)
	expectErrorContains(t, err, "SELECT pg_advisory_unlock")
}

func TestComputeMigrationPlanFailure(t *testing.T) {
	bq := BadQueryer{}
	withEachDialect(t, func(t *testing.T, d Dialect) {
		migrator := NewMigrator(WithDialect(d))
		_, err := migrator.computeMigrationPlan(bq, []*Migration{})
		expectErrorContains(t, err, "FAIL: SELECT id, checksum, execution_time_in_millis, applied_at")
	})
}

func expectErrorContains(t *testing.T, err error, contains string) {
	if err == nil {
		t.Errorf("Expected an error string containing '%s', got nil", contains)
	} else if !strings.Contains(err.Error(), contains) {
		t.Errorf("Expected an error string containing '%s', got '%s' instead", contains, err.Error())
	}
}
