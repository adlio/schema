package schema

import (
	"fmt"
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
	return strings.TrimSuffix(path.Base(filename), path.Ext(filename))
}

// MigrationsFromDirectoryPath retrieves a slice of Migrations from the
// contents of the directory. Only .sql files are read
func MigrationsFromDirectoryPath(dirPath string) (migrations []*Migration, err error) {
	migrations = make([]*Migration, 0)

	dirPath, err = filepath.Abs(dirPath)
	if err != nil {
		return
	}

	if _, err = os.Stat(dirPath); os.IsNotExist(err) {
		err = fmt.Errorf("Directory '%s' does not exist", dirPath)
		return
	}

	filenames, err := filepath.Glob(path.Join(dirPath, "*.sql"))
	if err != nil {
		return
	}

	for _, filename := range filenames {
		var content []byte
		content, err = ioutil.ReadFile(filename)
		if err != nil {
			return
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
