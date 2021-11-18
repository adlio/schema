# Schema - Embedded Database Migration Library for Go

An opinionated, embeddable library for tracking and application modifications
to your Go application's database schema.

[![Build Status](https://travis-ci.org/adlio/schema.svg?branch=main)](https://travis-ci.org/adlio/schema)
[![Go Report Card](https://goreportcard.com/badge/github.com/adlio/schema)](https://goreportcard.com/report/github.com/adlio/schema)
[![codecov](https://codecov.io/gh/adlio/schema/branch/main/graph/badge.svg)](https://codecov.io/gh/adlio/schema)
[![GoDoc](https://godoc.org/github.com/adlio/schema?status.svg)](https://godoc.org/github.com/adlio/schema)

## Package Opinions

There are many other schema migration tools. This one exists because of a
particular set of opinions:

1. Database credentials are runtime configuration details, but database
   schema is a **build-time applicaton dependency**, which means it should be
   "compiled in" to the build, and should not rely on external tools.
2. Using an external command-line tool for schema migrations needlessly
   complicates testing and deployment.
3. Sequentially-numbered integer migration IDs will create too many unnecessary
   schema collisions on a distributed, asynchronously-communicating team.
4. SQL is the best language to use to specify changes to SQL schemas.
5. "Down" migrations add needless complication, aren't often used, and are
   tedious to properly test when they are used. In the unlikely event you need
   to migrate backwards, it's possible to write the "rollback" migration as
   a separate "up" migration.
6. Deep dependency chains should be avoided, especially in a compiled
   binary. We don't want to import an ORM into our binaries just to get SQL
   the features of this package. The `schema` package imports only
   [standard library packages](https://godoc.org/github.com/adlio/schema?imports)
   (**NOTE** \*We do import `ory/dockertest` in our tests).
7. Storing raw SQL as strings inside `.go` files is an acceptable trade-off
   for the above. (For users who depend on `.sql` files, bolt-on integrations
   of go-bindata, go-rice or similar binary embedders are possible).

## Supported Databases

This package was extracted from a PostgreSQL project, so that's all that's
tested at the moment, but all the databases below should be easy to add
with a [contribution](#contributions):

- [x] PostgreSQL
- [ ] MySQL (open a Pull Request)
- [ ] SQLite (open a Pull Request)
- [ ] SQL Server (open a Pull Request)

## Roadmap

- [x] Basic feature set for PostgreSQL
- [x] Continuous integration tests / Code coverage
- [x] Basic Documentation: basic overview, usage documentation
- [ ] Add a validation pass inside `Apply()` to throw an error when checksums or
      IDs of previously-run migrations appear to have been changed or when
      problematic migration IDs are being used.
- [ ] Enhancements to facilitate asset embedding tools like
      [go-rice](https://github.com/GeertJohan/go.rice) or
      [packr](https://github.com/gobuffalo/packr) to get syntax highlighting for
      external `.sql` files which are embedded only at build time (or clear
      documentation to explain how they can be used without changing `schema`).
- [ ] Support for additional databases.

## Usage Instructions

Create a `schema.Migrator` in your bootstrap/config/database connection code,
then call its `Apply()` method with your database connection and a slice of
`*schema.Migration` structs. Like so:

    db, err := sql.Open(...) // Or however you get a *sql.DB

    migrator := schema.NewMigrator()
    migrator.Apply(db, []*schema.Migration{
      &schema.Migration{
        ID: "2019-09-24 Create Albums",
        Script: `
        CREATE TABLE albums (
          id SERIAL PRIMARY KEY,
          title CHARACTER VARYING (255) NOT NULL
        )
        `
      }
    })

The `.Apply()` function figures out which of the supplied Migrations have not
yet been executed in the database (based on the ID), and executes the `Script`
for each in **alphabetical order by ID**. This procedure means its OK to call
`.Apply()` on the same Migrator with a different set of Migrations each time
(which you might do if you want to avoid the ugliness of one giant migrations.go
file with hundreds of lines of embedded SQL in it).

The `NewMigrator()` function accepts option arguments to customize the dialect
and the name of the migration tracking table. By default, the tracking table
will be set to `schema.DefaultTableName` (`schema_migrations`). To change it
to `my_migrations`:

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

## Rules of Applying Migrations

1.  **Never, ever change** the `ID` or `Script` of a Migration which has already
    been executed on your database. If you've made a mistake, you'll need to correct
    it in a subsequent migration.
2.  Use a consistent, but descriptive format for migration `ID`s. Your format
    Consider
    prefixing them with today's timestamp. Examples:

            ID: "2019-01-01T13:45:00 Creates Users"
            ID: "2001-12-18 001 Changes the Default Value of User Affiliate ID"

        Do not use simple sequentialnumbers like `ID: "1"`.

## Migration Ordering

Migrations **are not** executed in the order they are specified in the slice.
They will be re-sorted alphabetically by their IDs before executing them.

## Inspecting the State of Applied Migrations

Call `migrator.GetAppliedMigrations(db)` to get info about migrations which
have been successfully applied.

## Contributions

... are welcome. Please include tests with your contribution. We've integrated
[dockertest](https://github.com/ory/dockertest) to automate the process of
creating clean test databases.

Before contributing, please read the [package opinions](#package-opinions)
section. If your contribution is in disagreement with those opinions, then
there's a good chance a different schema migration tool is more appropriate.

## Version History

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
