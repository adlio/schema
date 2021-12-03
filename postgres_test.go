package schema

import (
	"testing"
)

// Interface verification that Postgres is a valid Dialect
var (
	_ Dialect = Postgres
	_ Locker  = Postgres
)

func TestPostgreSQLQuotedTableName(t *testing.T) {
	type qtnTest struct {
		schema, table string
		expected      string
	}
	tests := []qtnTest{
		{"public", "users", `"public"."users"`},
		{"schema.with.dot", "table.with.dot", `"schema.with.dot"."table.with.dot"`},
		{`public"`, `"; DROP TABLE users`, `"public"""."""DROPTABLEusers"`},
	}
	for _, test := range tests {
		actual := Postgres.QuotedTableName(test.schema, test.table)
		if actual != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, actual)
		}
	}
}

func TestPostgreSQLQuotedIdent(t *testing.T) {
	table := map[string]string{
		"":                  "",
		"MY_TABLE":          `"MY_TABLE"`,
		"users_roles":       `"users_roles"`,
		"table.with.dot":    `"table.with.dot"`,
		`table"with"quotes`: `"table""with""quotes"`,
	}
	for ident, expected := range table {
		actual := Postgres.QuotedIdent(ident)
		if expected != actual {
			t.Errorf("Expected %s, got %s", expected, actual)
		}
	}
}
