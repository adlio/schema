package schema

import (
	"embed"
	"errors"
	"io/fs"
	"os"
	"testing"
	"testing/fstest"
)

//go:embed example-migrations
var exampleMigrations embed.FS

func TestMigrationsFromEmbedFS(t *testing.T) {
	migrations, err := FSMigrations(exampleMigrations, "example-migrations")
	if err != nil {
		t.Error(err)
	}

	expectedCount := 2
	if len(migrations) != expectedCount {
		t.Errorf("Expected %d migrations, got %d", expectedCount, len(migrations))
	}

	SortMigrations(migrations)
	expectID(t, migrations[0], "2019-01-01 0900 Create Users")
	expectScriptMatch(t, migrations[0], `^CREATE TABLE users`)
	expectID(t, migrations[1], "2019-01-03 1000 Create Affiliates")
	expectScriptMatch(t, migrations[1], `^CREATE TABLE affiliates`)
}

func TestMigrationsWithInvalidGlob(t *testing.T) {
	_, err := FSMigrations(exampleMigrations, "/a/path[]with/bad/glob/pattern")
	expectErrorContains(t, err, "/a/path[]with/bad/glob/pattern")
}

func TestFSMigrationsWithInvalidFiles(t *testing.T) {
	testfs := fstest.MapFS{
		"invalid-migrations": &fstest.MapFile{
			Mode: fs.ModeDir,
		},
		"invalid-migrations/real.sql": &fstest.MapFile{
			Data: []byte("File contents"),
		},
		"invalid-migrations/fake.sql": nil,
	}
	_, err := FSMigrations(testfs, "invalid-migrations")
	expectErrorContains(t, err, "fake.sql")
}

func TestMigrationFromFilePath(t *testing.T) {
	migration, err := MigrationFromFilePath("./example-migrations/2019-01-01 0900 Create Users.sql")
	if err != nil {
		t.Error(err)
	}
	expectID(t, migration, "2019-01-01 0900 Create Users")
	expectScriptMatch(t, migration, `^CREATE TABLE users`)
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
	expectID(t, migration, "2019-01-01 0900 Create Users")
	expectScriptMatch(t, migration, `^CREATE TABLE users`)
}

func TestMigrationsFromDirectoryPath(t *testing.T) {
	migrations, err := MigrationsFromDirectoryPath("./example-migrations")
	if err != nil {
		t.Error(err)
	}
	SortMigrations(migrations)
	expectID(t, migrations[0], "2019-01-01 0900 Create Users")
	expectID(t, migrations[1], "2019-01-03 1000 Create Affiliates")
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
