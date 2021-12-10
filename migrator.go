package schema

import (
	"context"
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

	ctx context.Context
}

// NewMigrator creates a new Migrator with the supplied
// options
func NewMigrator(options ...Option) *Migrator {
	m := Migrator{
		TableName: DefaultTableName,
		Dialect:   Postgres,
		ctx:       context.Background(),
	}
	for _, opt := range options {
		m = opt(m)
	}
	return &m
}

// QuotedTableName returns the dialect-quoted fully-qualified name for the
// migrations tracking table
func (m *Migrator) QuotedTableName() string {
	return m.Dialect.QuotedTableName(m.SchemaName, m.TableName)
}

// Apply takes a slice of Migrations and applies any which have not yet
// been applied against the provided database. Apply can be re-called
// sequentially with the same Migrations and different databases, but it is
// not threadsafe... if concurrent applies are desired, multiple Migrators
// should be used.
func (m *Migrator) Apply(db DB, migrations []*Migration) (err error) {
	// Reset state to begin the migration
	if db == nil {
		return ErrNilDB
	}

	if len(migrations) == 0 {
		return nil
	}

	if m.ctx == nil {
		m.ctx = context.Background()
	}

	// Obtain a concrete connection to the database which will be closed
	// at the conclusion of Apply()
	conn, err := db.Conn(m.ctx)
	if err != nil {
		return err
	}
	defer func() { err = coalesceErrs(err, conn.Close()) }()

	// If the database supports locking, obtain a lock around this migrator's
	// table name with a deferred unlock. Go's defers run LIFO, so this deferred
	// unlock will happen before the deferred conn.Close()
	err = m.lock(conn)
	if err != nil {
		return err
	}
	defer func() { err = coalesceErrs(err, m.unlock(conn)) }()

	tx, err := conn.BeginTx(m.ctx, nil)
	if err != nil {
		return err
	}

	err = m.Dialect.CreateMigrationsTable(m.ctx, tx, m.QuotedTableName())
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	err = m.run(tx, migrations)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	err = tx.Commit()

	return err
}

func (m *Migrator) lock(tx Queryer) error {
	if l, isLocker := m.Dialect.(Locker); isLocker {
		err := l.Lock(m.ctx, tx, m.QuotedTableName())
		if err != nil {
			return err
		}
		m.log(fmt.Sprintf("Locked %s at %s", m.QuotedTableName(), time.Now().Format(time.RFC3339Nano)))
	}
	return nil
}

func (m *Migrator) unlock(tx Queryer) error {
	if l, isLocker := m.Dialect.(Locker); isLocker {
		err := l.Unlock(m.ctx, tx, m.QuotedTableName())
		if err != nil {
			return err
		}
		m.log(fmt.Sprintf("Unlocked %s at %s", m.QuotedTableName(), time.Now().Format(time.RFC3339Nano)))
	}
	return nil
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

func (m *Migrator) run(tx Queryer, migrations []*Migration) error {
	if tx == nil {
		return ErrNilDB
	}

	plan, err := m.computeMigrationPlan(tx, migrations)
	if err != nil {
		return err
	}

	for _, migration := range plan {
		err = m.runMigration(tx, migration)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Migrator) runMigration(tx Queryer, migration *Migration) error {
	startedAt := time.Now()
	_, err := tx.ExecContext(m.ctx, migration.Script)
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

	applied := AppliedMigration{}
	applied.ID = migration.ID
	applied.Script = migration.Script
	applied.ExecutionTimeInMillis = ms
	applied.AppliedAt = startedAt
	return m.Dialect.InsertAppliedMigration(m.ctx, tx, m.QuotedTableName(), &applied)
}

func (m *Migrator) log(msgs ...interface{}) {
	if m.Logger != nil {
		m.Logger.Print(msgs...)
	}
}

func coalesceErrs(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}
