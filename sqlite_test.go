package schema

import "testing"

// Interface verification that SQLite is a valid Dialect
var (
	_ Dialect = SQLite
)

func TestSQLiteQuotedTableName(t *testing.T) {
	table := map[string]string{
		"":                    "",
		"MY_TABLE":            `"MY_TABLE"`,
		"users_roles":         `"users_roles"`,
		"table.with.dot":      `"table.with.dot"`,
		`table"with"quotes`:   `"table""with""quotes"`,
		`"; DROP TABLE users`: `"""DROPTABLEusers"`,
	}
	for ident, expected := range table {
		actual := SQLite.QuotedTableName("", ident)
		if expected != actual {
			t.Errorf("Expected %s, got %s", expected, actual)
		}
		actual = SQLite.QuotedTableName(ident, "")
		if expected != actual {
			t.Errorf("Expected %s, got %s when table came first", expected, actual)
		}
	}
}
