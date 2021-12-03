package schema

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

	Lock(db Queryer, tableName string) error
	Unlock(db Queryer, tableName string) error
}
