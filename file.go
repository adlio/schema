package schema

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// MigrationIDFromFilename removes directory paths and extensions
// from the filename to make a friendlier Migration ID
//
func MigrationIDFromFilename(filename string) string {
	return strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
}

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

// MigrationsFromDirectoryPath retrieves a slice of Migrations from the
// contents of the directory. Only .sql files are read
func MigrationsFromDirectoryPath(dirPath string) (migrations []*Migration, err error) {
	migrations = make([]*Migration, 0)

	// Assemble a glob of the .sql files in the directory. This can
	// only fail if the dirPath itself contains invalid glob characters
	filenames, err := filepath.Glob(filepath.Join(dirPath, "*.sql"))
	if err != nil {
		return migrations, err
	}

	// Friendly failure: if the user provides a valid-looking, but nonexistent
	// directory, we want to error instead of returning an empty set
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return migrations, fmt.Errorf("migrations directory does not exist: %w", err)
	}

	for _, filename := range filenames {
		migration, err := MigrationFromFilePath(filename)
		if err != nil {
			return migrations, err
		}
		migrations = append(migrations, migration)
	}
	return
}

// MigrationFromFilePath creates a Migration from a path on disk
func MigrationFromFilePath(filename string) (migration *Migration, err error) {
	migration = &Migration{}
	migration.ID = MigrationIDFromFilename(filename)
	contents, err := ioutil.ReadFile(path.Clean(filename))
	if err != nil {
		return migration, fmt.Errorf("failed to read migration from '%s': %w", filename, err)
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
