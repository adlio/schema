package schema

import (
	"testing"
)

func TestGetAppliedMigrationsErrorsWhenNoneExist(t *testing.T) {
	db := connectDB(t, "postgres11")
	migrator := makeTestMigrator()
	migrations, err := migrator.GetAppliedMigrations(db)
	if err == nil {
		t.Error("Expected an error. Got none.")
	}
	if len(migrations) > 0 {
		t.Error("Expected empty list of applied migrations")
	}
}
