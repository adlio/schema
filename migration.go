package schema

import (
	"sort"
)

// Migration is a yet-to-be-run change to the schema
type Migration struct {
	ID     string
	Script string
}

// SortMigrations sorts a slice of migrations by their IDs
func SortMigrations(migrations []*Migration) {
	// Adjust execution order so that we apply by ID
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].ID < migrations[j].ID
	})
}
