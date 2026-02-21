package schema

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	// MSSQL Driver
	_ "github.com/microsoft/go-mssqldb"
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

// mssqlBadQueryer implements the Queryer interface but fails on specific queries
type mssqlBadQueryer struct {
	failOnQuery string
	scanError   bool
}

func (bq mssqlBadQueryer) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return nil, nil
}

func (bq mssqlBadQueryer) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if strings.Contains(query, bq.failOnQuery) {
		return nil, fmt.Errorf("query failed: %s", query)
	}
	return nil, nil
}

func TestMSSQLUnlockQueryError(t *testing.T) {
	bq := mssqlBadQueryer{failOnQuery: "APPLOCK_MODE"}
	err := MSSQL.Unlock(context.Background(), bq, "test_table")
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if !strings.Contains(err.Error(), "error checking lock status") {
		t.Errorf("Expected error about lock status, got: %s", err)
	}
}

func TestMSSQLGetLockModeScanError(t *testing.T) {
	// Test that getLockMode handles scan errors properly
	// We use sqlmock to return rows that will cause a scan error
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Return a row with wrong type that will fail to scan into string
	rows := sqlmock.NewRows([]string{"lock_mode"}).AddRow(nil)
	mock.ExpectQuery("SELECT APPLOCK_MODE").WillReturnRows(rows)

	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	_, err = mssqlDialect{}.getLockMode(context.Background(), conn, 12345)
	if err == nil {
		t.Error("Expected error from scan, got nil")
	}
	if !strings.Contains(err.Error(), "error scanning lock mode") {
		t.Errorf("Expected scan error, got: %s", err)
	}
}

func TestMSSQLCreateMigrationsTableConcurrentError(t *testing.T) {
	// Test that concurrent table creation errors are ignored
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Simulate the "object already exists" error from concurrent creation
	mock.ExpectExec("IF NOT EXISTS").WillReturnError(
		fmt.Errorf("There is already an object named 'schema_migrations' in the database"),
	)

	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	err = mssqlDialect{}.CreateMigrationsTable(context.Background(), conn, "[schema_migrations]")
	if err != nil {
		t.Errorf("Expected nil error for concurrent creation, got: %s", err)
	}
}
