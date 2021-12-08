package schema

import "context"

// Dialect defines the minimal interface for a database dialect. All dialects
// must implement functions to create the migrations table, get all applied
// migrations, insert a new migration tracking record, and perform escaping
// for the tracking table's name
type Dialect interface {
	QuotedTableName(schemaName, tableName string) string

	CreateMigrationsTable(ctx context.Context, tx Queryer, tableName string) error
	GetAppliedMigrations(ctx context.Context, tx Queryer, tableName string) (applied []*AppliedMigration, err error)
	InsertAppliedMigration(ctx context.Context, tx Queryer, tableName string, migration *AppliedMigration) error
}

// Locker defines an optional Dialect extension for obtaining and releasing
// a global database lock during the running of migrations. This feature is
// supported by PostgreSQL and MySQL, but not SQLite.
type Locker interface {
	Lock(ctx context.Context, tx Queryer, tableName string) error
	Unlock(ctx context.Context, tx Queryer, tableName string) error
}
