package schema

import "database/sql"

// Dialect defines the interface for a database dialect.
// All interface functions take the customized table name
// as input and return a SQL statement with placeholders
// appropriate to the database.
//
type Dialect interface {
	QuotedTableName(schemaName, tableName string) string
	CreateSQL(tableName string) string
	SelectSQL(tableName string) string
	InsertSQL(tableName string) string
}

// Locking is achieved by implementing at least one of the
// Locker interfaces. If the database natively supports
// locking through SQL, the SQLLocker is simpler. If neither
// interface is present a panic will occur.

// Locker defines an interface that implements locking.
type Locker interface {
	Lock(db *sql.DB) error
	Unlock(db *sql.DB) error
}

// SQLLocker defines an interface that implements locking
// using a single SQL statement.
type SQLLocker interface {
	LockSQL(tableName string) string
	UnlockSQL(tableName string) string
}
