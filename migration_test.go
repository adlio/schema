package schema

import (
	"os"
	"testing"
)

func TestMigrationFromFilePath(t *testing.T) {
	migration, err := MigrationFromFilePath("./example-migrations/2019-01-01 0900 Create Users.sql")
	if migration.Script != "CREATE TABLE users (id INTEGER NOT NULL PRIMARY KEY);" {
		t.Error("Failed to get correct contents of migration")
	}
	if err != nil {
		t.Error(err)
	}
}

func TestMigrationFromFile(t *testing.T) {
	file, err := os.Open("./example-migrations/2019-01-01 0900 Create Users.sql")
	if err != nil {
		t.Error(err)
	}
	migration, err := MigrationFromFile(file)
	if err != nil {
		t.Error(err)
	}
	if migration.ID != "2019-01-01 09:00 Create Users" {
		t.Errorf("Incorrect ID: %s", migration.ID)
	}
	if migration.Script != "CREATE TABLE users (id INTEGER NOT NULL PRIMARY KEY);" {
		t.Errorf("Incorrect Script: %s", migration.Script)
	}
}

func TestMigrationsFromDirectoryPath(t *testing.T) {
	migrations, err := MigrationsFromDirectoryPath("./example-migrations")
	SortMigrations(migrations)
	if err != nil {
		t.Error(err)
	}
	if migrations[0].ID != "2019-01-01 0900 Create Users" {
		t.Errorf("Incorrect ID: %s", migrations[0].ID)
	}
	if migrations[1].ID != "2019-01-03 1000 Create Affiliates" {
		t.Errorf("Incorrect ID: %s", migrations[1].ID)
	}
}
