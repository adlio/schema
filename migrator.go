package schema

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Migrator is an instance customized to perform migrations on a particular
// database against a particular tracking table and with a particular dialect
// defined.
type Migrator struct {
	SchemaName string
	TableName  string
	Dialect    Dialect
	Logger     Logger

	// err holds the last error which occurred at any step of the migration
	// process
	err error
}

// NewMigrator creates a new Migrator with the supplied
// options
func NewMigrator(options ...Option) Migrator {
	m := Migrator{
		TableName: DefaultTableName,
		Dialect:   Postgres,
	}
	for _, opt := range options {
		m = opt(m)
	}
	return m
}

// Apply takes a slice of Migrations and applies any which have not yet
// been applied
func (m *Migrator) Apply(db Connection, migrations []*Migration) (err error) {
	if db == nil {
		return ErrNilDB
	}

	m.err = nil

	m.lock(db)
	defer m.unlock(db)

	m.transaction(db, func(tx Queryer) {
		m.createMigrationsTable(tx)
		m.run(tx, migrations)
	})

	return m.err
}

// QuotedTableName returns the dialect-quoted fully-qualified name for the
// migrations tracking table
func (m *Migrator) QuotedTableName() string {
	return m.Dialect.QuotedTableName(m.SchemaName, m.TableName)
}

func (m *Migrator) createMigrationsTable(tx Queryer) {
	if m.err != nil {
		// Abort if Migrator already had an error
		return
	}
	_, m.err = tx.Exec(m.Dialect.CreateSQL(m.QuotedTableName()))
}

func (m *Migrator) lock(tx Queryer) {
	if m.err != nil {
		// Abort if Migrator already had an error
		return
	}
	if l, isLocker := m.Dialect.(Locker); isLocker {
		query := l.LockSQL(m.TableName)
		_, err := tx.Exec(query)
		if err == nil {
			m.log("Locked at ", time.Now().Format(time.RFC3339Nano))
		} else {
			m.err = err
		}
	}
}

func (m *Migrator) unlock(tx Queryer) {
	if l, isLocker := m.Dialect.(Locker); isLocker {
		query := l.UnlockSQL(m.TableName)
		_, err := tx.Exec(query)
		if err == nil {
			m.log("Unlocked at ", time.Now().Format(time.RFC3339Nano))
		} else if m.err == nil {
			// Only set the migrator error if we're not overwriting an
			// earlier error
			m.err = err
		}
	}
}

func (m *Migrator) computeMigrationPlan(tx Queryer, toRun []*Migration) (plan []*Migration, err error) {
	applied, err := m.GetAppliedMigrations(tx)
	if err != nil {
		return plan, err
	}

	plan = make([]*Migration, 0)
	for _, migration := range toRun {
		if _, exists := applied[migration.ID]; !exists {
			plan = append(plan, migration)
		}
	}

	SortMigrations(plan)
	return plan, err
}

func (m *Migrator) run(tx Queryer, migrations []*Migration) {
	if m.err != nil {
		// Abort if Migrator already had an error
		return
	}

	if tx == nil {
		m.err = ErrNilDB
		return
	}

	var plan []*Migration
	plan, m.err = m.computeMigrationPlan(tx, migrations)
	if m.err != nil {
		return
	}

	for _, migration := range plan {
		m.err = m.runMigration(tx, migration)
		if m.err != nil {
			return
		}
	}
}

func (m *Migrator) runMigration(tx Queryer, migration *Migration) (err error) {
	startedAt := time.Now()
	_, err = tx.Exec(migration.Script)
	if err != nil {
		return fmt.Errorf("Migration '%s' Failed:\n%w", migration.ID, err)
	}

	executionTime := time.Since(startedAt)
	m.log(fmt.Sprintf("Migration '%s' applied in %s\n", migration.ID, executionTime))

	ms := executionTime.Milliseconds()
	if ms == 0 && executionTime.Microseconds() > 0 {
		// Avoid rounding down to 0 for very, very fast migrations
		ms = 1
	}

	_, err = tx.Exec(
		m.Dialect.InsertSQL(m.QuotedTableName()),
		migration.ID,
		migration.MD5(),
		ms,
		startedAt,
	)
	return err
}

// transaction wraps the supplied function in a transaction with the supplied
// database connection
//
func (m *Migrator) transaction(db Transactor, f func(Queryer)) {
	if db == nil {
		m.err = ErrNilDB
		return
	}

	if m.err != nil {
		// Abort if Migrator already had an error
		return
	}

	var tx *sql.Tx
	tx, m.err = db.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelSerializable})
	if m.err != nil {
		return
	}

	defer func() {
		if p := recover(); p != nil {
			switch p := p.(type) {
			case error:
				m.err = p
			default:
				m.err = fmt.Errorf("%s", p)
			}
		}
		if m.err != nil {
			if tx != nil {
				_ = tx.Rollback()
			}
			return
		} else if tx != nil {
			m.err = tx.Commit()
		}
	}()

	f(tx)
}

func (m *Migrator) log(msgs ...interface{}) {
	if m.Logger != nil {
		m.Logger.Print(msgs...)
	}
}
