package schema

import (
	"crypto/md5"
	"database/sql"
	"fmt"
	"time"
)

// Migrator is an instance customized to perform migrations on a particular
// against a particular tracking table and with a particular dialect
// defined.
type Migrator struct {
	SchemaName string
	TableName  string
	Dialect    Dialect
	Logger     Logger
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
func (m Migrator) Apply(db *sql.DB, migrations []*Migration) error {
	err := m.lock(db)
	if err != nil {
		return err
	}

	err = m.createMigrationsTable(db)
	if err != nil {
		return err
	}

	err = transaction(db, func(tx *sql.Tx) error {
		_, err := tx.Exec(m.Dialect.LockSQL(m.QuotedTableName()))
		if err != nil {
			return err
		}

		applied, err := m.GetAppliedMigrations(tx)
		if err != nil {
			return err
		}

		plan := make([]*Migration, 0)
		for _, migration := range migrations {
			if _, exists := applied[migration.ID]; !exists {
				plan = append(plan, migration)
			}
		}

		SortMigrations(plan)

		for _, migration := range plan {
			err = m.runMigration(tx, migration)
			if err != nil {
				return err
			}
		}

		return nil
	})

	_ = m.unlock(db)
	return err
}

// QuotedTableName returns the dialect-quoted fully-qualified name for the
// migrations tracking table
func (m Migrator) QuotedTableName() string {
	return m.Dialect.QuotedTableName(m.SchemaName, m.TableName)
}

func (m Migrator) createMigrationsTable(db *sql.DB) (err error) {
	return transaction(db, func(tx *sql.Tx) error {
		_, err := tx.Exec(m.Dialect.CreateSQL(m.QuotedTableName()))
		return err
	})
}

func (m Migrator) lock(db *sql.DB) (err error) {
	if db == nil {
		return ErrNilDB
	}
	_, err = db.Exec(m.Dialect.LockSQL(m.TableName))
	m.log("Locked at ", time.Now().Format(time.RFC3339Nano))
	return err
}

func (m Migrator) unlock(db *sql.DB) (err error) {
	if db == nil {
		return ErrNilDB
	}
	_, err = db.Exec(m.Dialect.UnlockSQL(m.TableName))
	m.log("Unlocked at ", time.Now().Format(time.RFC3339Nano))
	return err
}

func (m Migrator) runMigration(tx *sql.Tx, migration *Migration) error {
	var (
		err      error
		checksum string
	)

	startedAt := time.Now()
	_, err = tx.Exec(migration.Script)
	if err != nil {
		return fmt.Errorf("Migration '%s' Failed:\n%w", migration.ID, err)
	}

	executionTime := time.Since(startedAt)
	m.log(fmt.Sprintf("Migration '%s' applied in %s\n", migration.ID, executionTime))

	checksum = fmt.Sprintf("%x", md5.Sum([]byte(migration.Script)))
	_, err = tx.Exec(
		m.Dialect.InsertSQL(m.QuotedTableName()),
		migration.ID,
		checksum,
		executionTime.Milliseconds(),
		startedAt,
	)
	return err
}

func (m Migrator) log(msgs ...interface{}) {
	if m.Logger != nil {
		m.Logger.Print(msgs...)
	}
}
