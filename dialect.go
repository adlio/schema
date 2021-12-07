package schema

import "context"

// Dialect defines the minimal interface for a database dialect.
// All interface functions take the customized table name
// as input and return a SQL statement with placeholders
// appropriate to the database.
//
type Dialect interface {
	QuotedTableName(schemaName, tableName string) string
	CreateSQL(tableName string) string
	GetAppliedMigrations(ctx context.Context, tx Queryer, tableName string) (applied []*AppliedMigration, err error)
	InsertSQL(tableName string) string
}

// Locker defines an optional Dialect extension for obtaining and releasing
// a global database lock during the running of migrations. This feature is
// supported by PostgreSQL and MySQL, but not SQLite.
type Locker interface {
	Lock(ctx context.Context, tx Queryer, tableName string) error
	Unlock(ctx context.Context, tx Queryer, tableName string) error
}
