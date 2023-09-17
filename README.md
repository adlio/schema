# Schema - Database Migrations for Go

An embeddable library for applying changes to your Go application's
`database/sql` schema.

[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=for-the-badge)](https://pkg.go.dev/github.com/adlio/schema)
[![CircleCI Build Status](https://img.shields.io/circleci/build/github/adlio/schema?style=for-the-badge)](https://dl.circleci.com/status-badge/redirect/gh/adlio/schema/tree/main)
[![Go Report Card](https://goreportcard.com/badge/github.com/adlio/schema?style=for-the-badge)](https://goreportcard.com/report/github.com/adlio/schema)
[![Code Coverage](https://img.shields.io/codecov/c/github/adlio/schema?style=for-the-badge)](https://codecov.io/gh/adlio/schema)

## Features

- Cloud-friendly design tolerates embedded use in clusters
- Supports migrations in embed.FS (requires go:embed in Go 1.16+)
- [Depends only on Go standard library](https://pkg.go.dev/github.com/adlio/schema?tab=imports) (Note that all go.mod dependencies are used only in tests)
- Unidirectional migrations (no "down" migration complexity)

# Usage Instructions

Create a `schema.Migrator` in your bootstrap/config/database connection code,
then call its `Apply()` method with your database connection and a slice of
`*schema.Migration` structs.

The `.Apply()` function figures out which of the supplied Migrations have not
yet been executed in the database (based on the ID), and executes the `Script`
for each in **alphabetical order by IDe**.

The `[]*schema.Migration` can be created manually, but the package
has some utility functions to make it easier to parse .sql files into structs
with the filename as the `ID` and the file contents as the `Script`.

## Using go:embed (requires Go 1.16+)

Go 1.16 added features to embed a directory of files into the binary as an
embedded filesystem (`embed.FS`).

Assuming you have a directory of SQL files called `my-migrations/` next to your
main.go file, you'll run something like this:

```go
//go:embed my-migrations
var MyMigrations embed.FS

func main() {
   db, err := sql.Open(...) // Or however you get a *sql.DB

   migrations, err := schema.FSMigrations(MyMigrations, "my-migrations/*.sql")
   migrator := schema.NewMigrator(schema.WithDialect(schema.MySQL))
   err = migrator.Apply(db, migrations)
}
```

The `WithDialect()` option accepts: `schema.MySQL`, `schema.Postgres`,
`schema.SQLite` or `schema.MSSQL`. These dialects all use only `database/sql`
calls, so you may have success with other databases which are SQL-compatible
with the above dialects.

You can also provide your own custom `Dialect`. See `dialect.go` for the
definition of the `Dialect` interface, and the optional `Locker` interface. Note
that `Locker` is critical for clustered operation to ensure that only one of
many processes is attempting to run migrations simultaneously.

## Using Inline Migration Structs

If you're running in an earlier version of Go, Migration{} structs will need to
be created manually:

```go
db, err := sql.Open(...)

migrator := schema.NewMigrator() // Postgres is the default Dialect
migrator.Apply(db, []*schema.Migration{
   &schema.Migration{
      ID: "2019-09-24 Create Albums",
      Script: `
      CREATE TABLE albums (
         id SERIAL PRIMARY KEY,
         title CHARACTER VARYING (255) NOT NULL
      )
      `
   },
})
```

## Constructor Options

The `NewMigrator()` function accepts option arguments to customize the dialect
and the name of the migration tracking table. By default, the tracking table
will be named `schema_migrations`. To change it
to `my_migrations` instead:

```go
migrator := schema.NewMigrator(schema.WithTableName("my_migrations"))
```

It is theoretically possible to create multiple Migrators and to use mutliple
migration tracking tables within the same application and database.

It is also OK for multiple processes to run `Apply` on identically configured
migrators simultaneously. The `Migrator` only creates the tracking table if it
does not exist, and then locks it to modifications while building and running
the migration plan. This means that the first-arriving process will **win** and
will perform its migrations on the database.

## Supported Databases

This package was extracted from a PostgreSQL project. Other databases have solid
automated test coverage, but should be considered somewhat experimental in
production use cases. [Contributions](#contributions) are welcome for
additional databases or feature enhancements / bug fixes.

- [x] PostgreSQL (database/sql driver only, see [adlio/pgxschema](https://github.com/adlio/pgxschema) if you use `jack/pgx`)
- [x] SQLite (thanks [kalafut](https://github.com/kalafut)!)
- [x] MySQL / MariaDB
- [x] SQL Server
- [ ] CockroachDB, Redshift, Snowflake, etc (open a Pull Request)

## Package Opinions

There are many other schema migration tools. This one exists because of a
particular set of opinions:

1. Database credentials are runtime configuration details, but database
   schema is a **build-time applicaton dependency**, which means it should be
   "compiled in" to the build, and should not rely on external tools.
2. Using an external command-line tool for schema migrations needlessly
   complicates testing and deployment.
3. SQL is the best language to use to specify changes to SQL schemas.
4. "Down" migrations add needless complication, aren't often used, and are
   tedious to properly test when they are used. In the unlikely event you need
   to migrate backwards, it's possible to write the "rollback" migration as
   a separate "up" migration.
5. Deep dependency chains should be avoided, especially in a compiled
   binary. We don't want to import an ORM into our binaries just to get SQL
   the features of this package. The `schema` package imports only
   [standard library packages](https://godoc.org/github.com/adlio/schema?imports)
   (**NOTE** \*We do import `ory/dockertest` in our tests).
6. Sequentially-numbered integer migration IDs will create too many unnecessary
   schema collisions on a distributed, asynchronously-communicating team
   (this is not yet strictly enforced, but may be later).

## Rules of Applying Migrations

1.  **Never, ever change** the `ID` (filename) or `Script` (file contents)
    of a Migration which has already been executed on your database. If you've
    made a mistake, you'll need to correct it in a subsequent migration.
2.  Use a consistent, but descriptive format for migration `ID`s/filenames.
    Consider prefixing them with today's timestamp. Examples:

         ID: "2019-01-01T13:45:00 Creates Users"
         ID: "2001-12-18 001 Changes the Default Value of User Affiliate ID"

    Do not use simple sequentialnumbers like `ID: "1"`.

## Migration Ordering

Migrations **are not** executed in the order they are specified in the slice.
They will be re-sorted alphabetically by their IDs before executing them.

## Contributions

... are welcome. Please include tests with your contribution. We've integrated
[dockertest](https://github.com/ory/dockertest) to automate the process of
creating clean test databases.

Before contributing, please read the [package opinions](#package-opinions)
section. If your contribution is in disagreement with those opinions, then
there's a good chance a different schema migration tool is more appropriate.

## Roadmap

- [x] Enhancements and documentation to facilitate asset embedding via go:embed
- [ ] Add a `Validate()` method to allow checking migration names for
      consistency and to detect problematic changes in the migrations list.
- [x] SQL Server support
- [ ] SQL Server support for the Locker interface to protect against simultaneous
      migrations from clusters of servers.

## Version History

### 1.3.4 - Apr 9, 2023

- Update downstream dependencies to address vulnerabilities in test dependencies.

### 1.3.3 - Jun 19, 2022

- Update downstream dependencies of ory/dockertest due to security issues.

### 1.3.0 - Mar 25, 2022

- Basic SQL Server support (no locking, not recommended for use in clusters)
- Improved support for running tests on ARM64 machines (M1 Macs)

### 1.2.3 - Dec 10, 2021

- BUGFIX: Restore the ability to chain NewMigrator().Apply

### 1.2.2 - Dec 9, 2021

- Add support for migrations in an embed.FS (`FSMigrations(filesystem fs.FS, glob string)`)
- Add MySQL/MariaDB support (experimental)
- Add SQLite support (experimental)
- Update go.mod to `go 1.17`.

### 1.1.14 - Nov 18, 2021

Security patches in upstream dependencies.

### 1.1.13 - May 22, 2020

Bugfix for error with advisory lock being held open. Improved test coverage for
simultaneous execution.

### 1.1.11 - May 19, 2020

Use a database-held lock for all migrations not just the initial table creation.

### 1.1.9 - May 17, 2020

Add the ability to attach a logger.

### 1.1.8 - Nov 24, 2019

Switch to `filepath` package for improved cross-platform filesystem support.

### 1.1.7 - Oct 1, 2019

Began using pg_advisory_lock() to prevent race conditions when multiple
processes/machines try to simultaneously create the migrations table.

### 1.1.1 - Sep 28, 2019

First published version.
