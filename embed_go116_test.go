//go:build go1.16
// +build go1.16

package schema

import (
	"embed"
	"io/fs"
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
