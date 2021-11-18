package schema

import (
	"errors"
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

func TestMigrationFromFilePathWithInvalidPath(t *testing.T) {
	_, err := MigrationFromFilePath("./example-migrations/nonexistent-file.sql")
	if err == nil {
		t.Errorf("Expected failure when reading from nonexistent file")
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
	if err == nil {
		t.Error("Expected an error trying to load migrations from a fake directory")
	}
	if len(migrations) > 0 {
		t.Errorf("Expected an empty list of migrations. Got %d", len(migrations))
	}
}

func TestMigrationsFromDirectoryPathThrowsErrorForInvalidGlob(t *testing.T) {
	_, err := MigrationsFromDirectoryPath("/a/path[]with/bad/glob/pattern")
	if err == nil {
		t.Error("Expected an error trying to load migrations from a fake directory")
	}
}

func TestMigrationsFromDirectoryPathThrowsErrorWithUnreadableFiles(t *testing.T) {
	err := os.Chmod("./unreadable-migrations/unreadable.sql", 0200)
	if err != nil {
		t.Error(err)
	}
	_, err = MigrationsFromDirectoryPath("./unreadable-migrations")
	if err == nil {
		t.Error("Expected a failure when trying to read unreadable file")
	}
	_ = os.Chmod("./unreadable-migrations/unreadable.sql", 0644) // #nosec
}

type failedReader int

func (fr failedReader) Name() string {
	return "fakeFile.sql"
}

func (fr failedReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("this reader always fails")
}

func TestMigrationFromFileWithUnreadableFile(t *testing.T) {
	var fr failedReader
	_, err := MigrationFromFile(fr)
	if err == nil {
		t.Error("Expected MigrationFromFile to fail when given erroneous file")
	}
}
