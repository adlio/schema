//go:build go1.16
// +build go1.16

package schema

import (
	"fmt"
	"io/fs"
)

// FSMigrations receives a filesystem (such as an embed.FS) and extracts all
// files matching the provided glob as Migrations, with the filename (without extension)
// being the ID and the file's contents being the Script.
//
// Example usage:
//
//	FSMigrations(embeddedFS, "my-migrations/*.sql")
func FSMigrations(filesystem fs.FS, glob string) (migrations []*Migration, err error) {
	migrations = make([]*Migration, 0)

	entries, err := fs.Glob(filesystem, glob)
	if err != nil {
		return migrations, fmt.Errorf("failed to process glob '%s' in embed.FS: %w", glob, err)
	}

	for _, entry := range entries {
		migration := &Migration{
			ID: MigrationIDFromFilename(entry),
		}
		data, err := fs.ReadFile(filesystem, entry)
		if err != nil {
			return migrations, err
		}
		migration.Script = string(data)
		migrations = append(migrations, migration)
	}
	return migrations, nil
}
