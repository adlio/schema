package schema

import (
	"os"
	"testing"
)

func TestMigrationIDFromFilename(t *testing.T) {
	const (
		sampleID = "2019-01-01 0900 Create Users"
	)
	tests := []struct {
		path string
		want string
		name string
	}{
		{"c:\\db\\migrations\\"+sampleID+".sql", sampleID, "Windows Path"},
		{"\\\\server\\db\\migrations\\"+sampleID+".sql", sampleID, "Windows UNC Path"},
		{"/db/migrations/"+sampleID+".sql", "2019-01-01 0900 Create Users", "Linux Path"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := MigrationIDFromFilename(test.path)
			if got != test.want {
				t.Errorf("%s ID Error, want \"%s\", got \"%s\"", test.name, test.want, got)
			}
		})
	}
}

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
	if migration.ID != "2019-01-01 0900 Create Users" {
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

func TestMigrationsFromDirectoryPathThrowsErrorForInvalidDirectory(t *testing.T) {
	migrations, err := MigrationsFromDirectoryPath("/a/totally/made/up/directory/path")
	if err != nil {
		t.Error("Expected an error trying to load migrations from a fake directory")
	}
	if len(migrations) > 0 {
		t.Errorf("Expected an empty list of migrations. Got %d", len(migrations))
	}
}
