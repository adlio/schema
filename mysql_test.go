package schema

import (
	"testing"
	"time"

	// MySQL Driver
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

func TestMySQLTimeScanner(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		v := mysqlTime{}
		err := v.Scan(nil)
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("Time In UTC", func(t *testing.T) {
		v := mysqlTime{}
		expected := time.Date(2021, 1, 1, 18, 19, 20, 0, time.UTC)
		src, _ := time.ParseInLocation("2006-01-02 15:04:05", "2021-01-01 18:19:20", time.UTC)
		err := v.Scan(src)
		if err != nil {
			t.Error(err)
		}
		assertZonesMatch(t, time.Now(), v.Value)
		if expected.Unix() != v.Value.Unix() {
			t.Errorf("Expected %s, got %s", expected.Format(time.RFC3339), v.Value.Format(time.RFC3339))
		}
	})

	t.Run("Invalid String Time", func(t *testing.T) {
		v := mysqlTime{}
		err := v.Scan("2000-13-45 99:45:23")
		if err == nil {
			t.Errorf("Expected error scanning invalid time")
		}
	})
}
