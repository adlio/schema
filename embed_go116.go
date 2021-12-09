//go:build go1.16
// +build go1.16

package schema

import (
	"fmt"
	"io/fs"
	"path/filepath"
)

// FSMigrations receives a filesystem (such as an embed.FS), and extracts all
// of the .sql-named files within the provided directory
func FSMigrations(filesystem fs.FS, dirName string) (migrations []*Migration, err error) {
	migrations = make([]*Migration, 0)

	entries, err := fs.Glob(filesystem, filepath.Join(dirName, "*.sql"))
	if err != nil {
		return migrations, fmt.Errorf("failed to process directory '%s' in embed.FS: %w", dirName, err)
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
