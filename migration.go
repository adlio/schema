package schema

import (
	"crypto/md5" // #nosec MD5 only being used to fingerprint script contents, not for encryption
	"fmt"
	"sort"
)

// Migration is a yet-to-be-run change to the schema. This is the type which
// is provided to Migrator.Apply to request a schema change.
type Migration struct {
	ID     string
	Script string
}

// MD5 computes the MD5 hash of the Script for this migration so that it
// can be uniquely identified later.
func (m *Migration) MD5() string {
	return fmt.Sprintf("%x", md5.Sum([]byte(m.Script))) // #nosec not being used cryptographically
}

// SortMigrations sorts a slice of migrations by their IDs
func SortMigrations(migrations []*Migration) {
	// Adjust execution order so that we apply by ID
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].ID < migrations[j].ID
	})
}
