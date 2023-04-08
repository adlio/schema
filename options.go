package schema

import "context"

// Option supports option chaining when creating a Migrator.
// An Option is a function which takes a Migrator and
// returns a Migrator with an Option modified.
type Option func(m Migrator) Migrator

// WithDialect builds an Option which will set the supplied
// dialect on a Migrator. Usage: NewMigrator(WithDialect(MySQL))
func WithDialect(dialect Dialect) Option {
	return func(m Migrator) Migrator {
		m.Dialect = dialect
		return m
	}
}

// WithTableName is an option which customizes the name of the schema_migrations
// tracking table. It can be called with either 1 or 2 string arguments. If
// called with 2 arguments, the first argument is assumed to be a schema
// qualifier (for example, WithTableName("public", "schema_migrations") would
// assign the table named "schema_migrations" in the the default "public"
// schema for Postgres)
func WithTableName(names ...string) Option {
	return func(m Migrator) Migrator {
		switch len(names) {
		case 0:
			// No-op if no customization was provided
		case 1:
			m.TableName = names[0]
		default:
			m.SchemaName = names[0]
			m.TableName = names[1]
		}
		return m
	}
}

// WithContext is an Option which sets the Migrator to run within the provided
// Context
func WithContext(ctx context.Context) Option {
	return func(m Migrator) Migrator {
		m.ctx = ctx
		return m
	}
}

// Logger is the interface for logging operations of the logger.
// By default the migrator operates silently. Providing a Logger
// enables output of the migrator's operations.
type Logger interface {
	Print(...interface{})
}

// WithLogger builds an Option which will set the supplied Logger
// on a Migrator. Usage: NewMigrator(WithLogger(logrus.New()))
func WithLogger(logger Logger) Option {
	return func(m Migrator) Migrator {
		m.Logger = logger
		return m
	}
}
