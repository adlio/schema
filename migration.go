package schema

import (
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Migration is a yet-to-be-run change to the schema
type Migration struct {
	ID     string
	Script string
}

// AppliedMigration is a schema change which was successfully
// completed
type AppliedMigration struct {
	Migration
	Checksum              string
	ExecutionTimeInMillis int
	AppliedAt             time.Time
}

// SortMigrations sorts a slice of migrations by their IDs
func SortMigrations(migrations []*Migration) {
	// Adjust execution order so that we apply by ID
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].ID < migrations[j].ID
	})
}

// GetAppliedMigrations retrieves all already-applied migrations in a map keyed
// by the migration IDs
//
func (m Migrator) GetAppliedMigrations(db Queryer) (applied map[string]*AppliedMigration, err error) {
	applied = make(map[string]*AppliedMigration)
	migrations := make([]*AppliedMigration, 0)

	rows, err := db.Query(m.Dialect.SelectSQL(m.QuotedTableName()))
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		migration := AppliedMigration{}
		err = rows.Scan(&migration.ID, &migration.Checksum, &migration.ExecutionTimeInMillis, &migration.AppliedAt)
		migrations = append(migrations, &migration)
	}
	for _, migration := range migrations {
		applied[migration.ID] = migration
	}
	return applied, err
}

// MigrationsFromDirectoryPath retrieves a slice of Migrations from the
// contents of the directory. Only .sql files are read
func MigrationsFromDirectoryPath(dirPath string) (migrations []*Migration, err error) {
	migrations = make([]*Migration, 0)
	filenames, err := filepath.Glob(path.Join(dirPath, "*.sql"))
	for _, filename := range filenames {
		content, err := ioutil.ReadFile(filename)
		if err != nil {
			return migrations, err
		}
		migration := &Migration{
			ID:     MigrationIDFromFilename(filename),
			Script: string(content),
		}
		migrations = append(migrations, migration)
	}
	return
}

// MigrationFromFilePath creates a Migration from a path on disk
func MigrationFromFilePath(filename string) (migration *Migration, err error) {
	migration = &Migration{}
	migration.ID = MigrationIDFromFilename(filename)
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		return migration, fmt.Errorf("Failed to read migration from '%s': %w", filename, err)
	}
	migration.Script = string(contents)
	return migration, err
}

// File wraps the standard library io.Read and os.File.Name methods
type File interface {
	Name() string
	Read(b []byte) (n int, err error)
}

// MigrationFromFile builds a migration by reading from an open File-like
// object. The migration's ID will be based on the file's name. The file
// will *not* be closed after being read.
func MigrationFromFile(file File) (migration *Migration, err error) {
	migration = &Migration{}
	migration.ID = MigrationIDFromFilename(file.Name())
	content, err := ioutil.ReadAll(file)
	if err != nil {
		return migration, err
	}
	migration.Script = string(content)
	return migration, err
}

// MigrationIDFromFilename removes directory paths and extensions
// from the filename to make a friendlier Migration ID
//
func MigrationIDFromFilename(filename string) string {
	return strings.TrimSuffix(path.Base(filename), path.Ext(filename))
}
