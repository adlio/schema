package schema

import (
	"regexp"
	"testing"
)

func TestMD5(t *testing.T) {
	testMigration := Migration{Script: "test"}
	expected := "098f6bcd4621d373cade4e832627b4f6"
	if testMigration.MD5() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, testMigration.MD5())
	}
}

func TestSortMigrations(t *testing.T) {
	migrations := []*Migration{
		{ID: "2020-01-01"},
		{ID: "2021-01-01"},
		{ID: "2000-01-01"},
	}
	expectedOrder := []string{"2000-01-01", "2020-01-01", "2021-01-01"}
	SortMigrations(migrations)
	for i, migration := range migrations {
		if migration.ID != expectedOrder[i] {
			t.Errorf("Expected migration #%d to be %s, got %s", i, expectedOrder[i], migration.ID)
		}
	}
}

func expectID(t *testing.T, migration *Migration, expectedID string) {
	if migration.ID != expectedID {
		t.Errorf("Expected Migration to have ID '%s', got '%s' instead", expectedID, migration.ID)
	}
}

func expectScriptMatch(t *testing.T, migration *Migration, regexpString string) {
	re, err := regexp.Compile(regexpString)
	if err != nil {
		t.Fatalf("Invalid regexp: '%s': %s", regexpString, err)
	}
	if !re.MatchString(migration.Script) {
		t.Errorf("Expected migration Script to match '%s', but it did not. Script was:\n%s", regexpString, migration.Script)
	}
}
