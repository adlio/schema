package schema

import (
	"testing"

	// MSSQL Driver
	_ "github.com/denisenkom/go-mssqldb"
)

// Interface verification that MSSQL is a valid Dialect
var (
	_ Dialect = MSSQL
)

func TestMSSQLQuotedTableName(t *testing.T) {
	type qtnTest struct {
		schema, table string
		expected      string
	}

	tests := []qtnTest{
		{"public", "ta[ble", `[public].[ta[ble]`},
		{"pu[b]lic", "ta[ble", `[pu[b]]lic].[ta[ble]`},
		{"tempdb", "users", `[tempdb].[users]`},
		{"schema.with.dot", "table.with.dot", `[schema.with.dot].[table.with.dot]`},
		{`public"`, `"; DROP TABLE users`, `[public"].["DROPTABLEusers]`},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.expected, func(t *testing.T) {
			actual := MSSQL.QuotedTableName(tc.schema, tc.table)
			if actual != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, actual)
			}
		})
	}

}

func TestMSSQLQuotedIdent(t *testing.T) {
	table := map[string]string{
		"":                    "",
		"MY_TABLE":            "[MY_TABLE]",
		"users_roles":         `[users_roles]`,
		"table.with.dot":      `[table.with.dot]`,
		`table"with"quotes`:   `[table"with"quotes]`,
		"table[with]brackets": "[table[with]]brackets]",
	}

	for ident, expected := range table {
		t.Run(expected, func(t *testing.T) {
			actual := MSSQL.QuotedIdent(ident)
			if expected != actual {
				t.Errorf("Expected %s, got %s", expected, actual)
			}
		})
	}
}
