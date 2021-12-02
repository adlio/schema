package schema

import (
	"testing"
)

func TestPostgres11CreateMigrationsTable(t *testing.T) {
	db := connectDB(t, "postgres11")
	migrator := NewMigrator(WithDialect(Postgres))
	migrator.createMigrationsTable(db)
	if migrator.err != nil {
		t.Errorf("Error occurred when creating migrations table: %s", migrator.err)
	}

	// Test that we can re-run it safely
	migrator.createMigrationsTable(db)
	if migrator.err != nil {
		t.Errorf("Calling createMigrationsTable a second time failed: %s", migrator.err)
	}
}

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

func TestPostgres11MultiStatementMigrations(t *testing.T) {
	db := connectDB(t, "postgres11")
	tableName := "musicdatabase_migrations"
	migrator := NewMigrator(WithDialect(Postgres), WithTableName(tableName))

	migrationSet1 := []*Migration{
		{
			ID: "2019-09-23 Create Artists and Albums",
			Script: `
		CREATE TABLE artists (
			id SERIAL PRIMARY KEY,
			name CHARACTER VARYING (255) NOT NULL DEFAULT ''
		);
		CREATE UNIQUE INDEX idx_artists_name ON artists (name);
		CREATE TABLE albums (
			id SERIAL PRIMARY KEY,
			title CHARACTER VARYING (255) NOT NULL DEFAULT '',
			artist_id INTEGER NOT NULL REFERENCES artists(id)
		);
		`,
		},
	}
	err := migrator.Apply(db, migrationSet1)
	if err != nil {
		t.Error(err)
	}

	err = migrator.Apply(db, migrationSet1)
	if err != nil {
		t.Error(err)
	}

	secondMigratorWithPublicSchema := NewMigrator(WithDialect(Postgres), WithTableName("public", tableName))
	migrationSet2 := []*Migration{
		{
			ID: "2019-09-24 Create Tracks",
			Script: `
		CREATE TABLE tracks (
			id SERIAL PRIMARY KEY,
			name CHARACTER VARYING (255) NOT NULL DEFAULT '',
			artist_id INTEGER NOT NULL REFERENCES artists(id),
			album_id INTEGER NOT NULL REFERENCES albums(id)
		);`,
		},
	}
	err = secondMigratorWithPublicSchema.Apply(db, migrationSet2)
	if err != nil {
		t.Error(err)
	}
}
