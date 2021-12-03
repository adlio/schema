package schema

import (
	"testing"

	_ "github.com/go-sql-driver/mysql"
)

// Interface verification that MySQL is a valid Dialect
var (
	_ Dialect = MySQL
	_ Locker  = MySQL
)

func TestMySQLQuotedTableName(t *testing.T) {
	type qtnTest struct {
		schema, table string
		expected      string
	}

	table := []qtnTest{
		{"public", "users", "`public`.`users`"},
		{"schema.with.dot", "table.with.dot", "`schema.with.dot`.`table.with.dot`"},
		{"schema`with`tick", "table`with`tick", "`schema``with``tick`.`table``with``tick`"},
	}

	for _, test := range table {
		actual := MySQL.QuotedTableName(test.schema, test.table)
		if actual != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, actual)
		}
	}
}

func TestMySQLQuotedIdent(t *testing.T) {
	table := map[string]string{
		"":                  "",
		"MY_TABLE":          "`MY_TABLE`",
		"users_roles":       "`users_roles`",
		"table.with.dot":    "`table.with.dot`",
		`table"with"quotes`: "`table\"with\"quotes`",
		"table`with`ticks":  "`table``with``ticks`",
	}

	for input, expected := range table {
		actual := MySQL.quotedIdent(input)
		if actual != expected {
			t.Errorf("Expected %s, got %s", expected, actual)
		}
	}
}
