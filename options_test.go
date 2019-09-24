package schema

import (
	"testing"
)

func TestWithTableNameOptionWithSchema(t *testing.T) {
	schema := "special"
	table := "my_migrations"
	m := NewMigrator(WithTableName(schema, table))
	if m.SchemaName != schema {
		t.Errorf("Expected SchemaName to be '%s'. Got '%s' instead.", schema, m.SchemaName)
	}
	if m.TableName != table {
		t.Errorf("Expected TableName to be '%s'. Got '%s' instead.", table, m.TableName)
	}
}
func TestWithTableNameOptionWithoutSchema(t *testing.T) {
	name := "terrible_migrations_table_name"
	m := NewMigrator(WithTableName(name))
	if m.SchemaName != "" {
		t.Errorf("Expected SchemaName to be blank. Got '%s' instead.", m.SchemaName)
	}
	if m.TableName != name {
		t.Errorf("Expected TableName to be '%s'. Got '%s' instead.", name, m.TableName)
	}
}

func TestDefaultTableName(t *testing.T) {
	name := "schema_migrations"
	m := NewMigrator()
	if m.SchemaName != "" {
		t.Errorf("Expected SchemaName to be blank by default. Got '%s' instead.", m.SchemaName)
	}
	if m.TableName != name {
		t.Errorf("Expected TableName to be '%s' by default. Got '%s' instead.", name, m.TableName)
	}
}

func TestDefaultDialect(t *testing.T) {
	m := NewMigrator()
	if m.Dialect != Postgres {
		t.Errorf("Expected Migrator to have Postgres Dialect by default. Got: %v", m.Dialect)
	}
}

func TestWithDialectOption(t *testing.T) {
	m := Migrator{Dialect: nil}
	if m.Dialect != nil {
		t.Errorf("Expected nil Dialect. Got '%v'", m.Dialect)
	}
	modifiedMigrator := WithDialect(Postgres)(m)
	if modifiedMigrator.Dialect != Postgres {
		t.Errorf("Expected modifiedMigrator to have Postgres dialect. Got '%v'.", modifiedMigrator.Dialect)
	}
	if m.Dialect != nil {
		t.Errorf("Expected Option to not modify the original Migrator's Dialect, but it changed it to '%v'.", m.Dialect)
	}
}
