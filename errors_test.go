package schema

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"
)

var (
	// ErrConnFailed indicates that the Conn() method failed (couldn't get a connection)
	ErrConnFailed = fmt.Errorf("connect failed")

	// ErrBeginFailed indicates that the Begin() method failed (couldn't start Tx)
	ErrBeginFailed = fmt.Errorf("begin failed")

	// ErrPriorFailure indicates a failure happened earlier in the Migrator Apply()
	ErrPriorFailure = fmt.Errorf("previous error")
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

func TestApplyWithBadDB(t *testing.T) {
	bd := BadDB{}
	migrator := Migrator{}
	err := migrator.Apply(bd, makeValidUnorderedMigrations())
	if err != ErrConnFailed {
		t.Errorf("Expected %v, got %v", ErrConnFailed, err)
	}
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

func TestNilTransaction(t *testing.T) {
	nt := Transactor(nil)
	withEachDialect(t, func(t *testing.T, d Dialect) {
		migrator := NewMigrator(WithDialect(d))
		migrator.transaction(nt, func(q Queryer) {})
		if !errors.Is(migrator.err, ErrNilDB) {
			t.Errorf("Expected ErrNilDB. Got %v", migrator.err)
		}
	})
}

// TestLockFailure ensures that each dialect and test database throws an
// expected error when the attempt to lock the database fails.
//
func TestLockFailure(t *testing.T) {
	bc := BadConnection{}

	withEachTestDB(t, func(t *testing.T, tdb *TestDB) {
		if _, isLocker := tdb.Dialect.(Locker); isLocker {
			migrator := makeTestMigrator(WithDialect(tdb.Dialect))
			migrator.lock(bc)
			if migrator.err == nil {
				t.Fatal("Expected error due to failed lock")
			}
		}
	})
}

// TestLockWithPriorFailure ensures that each dialect and test database will
// report any prior migrator.err if one exists before attempting to lock the
// database
func TestLockWithPriorFailure(t *testing.T) {
	bc := BadConnection{}

	withEachTestDB(t, func(t *testing.T, tdb *TestDB) {
		migrator := makeTestMigrator(WithDialect(tdb.Dialect))
		migrator.err = ErrPriorFailure
		migrator.lock(bc)
		if migrator.err != ErrPriorFailure {
			t.Errorf("Expected error %v. Got %v", ErrPriorFailure, migrator.err)
		}
	})
}

// TestUnlockFailure ensures that each dialect and test database will report
// a failure in the Unlock() step after the Lock() step succeeded.
//
func TestUnlockFailure(t *testing.T) {
	bc := BadConnection{}
	withEachTestDB(t, func(t *testing.T, tdb *TestDB) {
		_, isLocker := tdb.Dialect.(Locker)
		if isLocker {
			db := tdb.Connect(t)
			defer func() { _ = db.Close() }()

			migrator := makeTestMigrator(WithDialect(tdb.Dialect))
			migrator.lock(db)
			if migrator.err != nil {
				t.Fatal(migrator.err)
			}

			migrator.unlock(bc)
			if migrator.err == nil {
				t.Error("Expected error due to failed unlock")
			}

			// Successfully unlock this time to leave the test database in a
			// happy state for other tests
			migrator.unlock(db)
		}
	})
}

func TestApplyWithPriorError(t *testing.T) {
	bc := BadConnection{}
	withEachDialect(t, func(t *testing.T, d Dialect) {
		migrator := NewMigrator(WithDialect(d))
		migrator.err = ErrPriorFailure
		migrator.transaction(bc, func(q Queryer) {})
		if migrator.err != ErrPriorFailure {
			t.Errorf("Expected error %v. Got %v", ErrPriorFailure, migrator.err)
		}
	})
}
func TestBeginTransactionFailure(t *testing.T) {
	bt := BadTransactor{}
	withEachDialect(t, func(t *testing.T, d Dialect) {
		migrator := NewMigrator(WithDialect(d))
		migrator.transaction(bt, func(q Queryer) {})
		if !errors.Is(migrator.err, ErrBeginFailed) {
			t.Errorf("Expected ErrBeginFailed, got %v", migrator.err)
		}
	})
}

func TestCreateMigrationsTableFailure(t *testing.T) {
	bq := BadQueryer{}
	withEachDialect(t, func(t *testing.T, d Dialect) {
		migrator := NewMigrator(WithDialect(d))
		migrator.err = ErrPriorFailure
		migrator.createMigrationsTable(bq)
		if migrator.err != ErrPriorFailure {
			t.Errorf("Expected error %v. Got %v.", ErrPriorFailure, migrator.err)
		}
	})
}

func TestComputeMigrationPlanFailure(t *testing.T) {
	bq := BadQueryer{}
	withEachDialect(t, func(t *testing.T, d Dialect) {
		migrator := NewMigrator(WithDialect(d))
		_, err := migrator.computeMigrationPlan(bq, []*Migration{})
		expectedContents := "FAIL: SELECT id, checksum, execution_time_in_millis, applied_at"
		if err == nil || !strings.Contains(err.Error(), expectedContents) {
			t.Errorf("Expected error msg with '%s'. Got '%v'.", expectedContents, err)
		}
	})
}

func TestLockWithPriorError(t *testing.T) {
	bc := BadConnection{}
	withEachDialect(t, func(t *testing.T, d Dialect) {
		migrator := NewMigrator(WithDialect(d))
		migrator.err = ErrPriorFailure
		migrator.lock(bc)
		if migrator.err != ErrPriorFailure {
			t.Errorf("Expected error %v. Got %v.", ErrPriorFailure, migrator.err)
		}
	})

func expectErrorContains(t *testing.T, err error, contains string) {
	if err == nil {
		t.Errorf("Expected an error string containing '%s', got nil", contains)
	} else if !strings.Contains(err.Error(), contains) {
		t.Errorf("Expected an error string containing '%s', got '%s' instead", contains, err.Error())
	}
}
