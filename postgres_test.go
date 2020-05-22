package schema

import (
	"strings"
	"testing"
	"time"
)

func TestPostgresLockSQL(t *testing.T) {
	name := `"schema_migrations"`

	sql := Postgres.LockSQL(name)
	if !strings.Contains(strings.ToLower(sql), "pg_advisory_lock") {
		t.Errorf("EXPECTED pg_advisory_lock:\n%s", sql)
	}
}
func TestPostgres11CreateMigrationsTable(t *testing.T) {
	db := connectDB(t, "postgres11")
	migrator := NewMigrator(WithDialect(Postgres))
	err := migrator.createMigrationsTable(db)
	if err != nil {
		t.Errorf("Error occurred when creating migrations table: %s", err)
	}

	// Test that we can re-run it safely
	err = migrator.createMigrationsTable(db)
	if err != nil {
		t.Errorf("Calling createMigrationsTable a second time failed: %s", err)
	}
}

func TestPostgres11MultiStatementMigrations(t *testing.T) {
	db := connectDB(t, "postgres11")
	tableName := time.Now().Format(time.RFC3339Nano)
	// tableName := "postgres_migrations"
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
