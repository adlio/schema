# Schema - Embedded Database Migration Library for Go

A package for tracking and application modifications to your Go application's
primary database schema.

## Package Opionions

There are many other schema migration tools. This one exists because of a
particular set of opinions

1. Database credentials are runtime configuration details, but database
schema is a build-time applicaton dependency, which means it should be
"compiled in" to the build.
2. Using an external command-line tool for schema migrations needlessly
complicates testing and deployment.
3. Sequentially-numbered migration IDs will create too many unnecessary
schema collisions on a distributed, asynchronously-communicating team.
4. SQL is the best language to use to specify changes to SQL schemas.
5. You shouldn't introduce an ORM for handling database schema changes.

## Supported Databases

This package was extracted from a Postgres project, so that's all that's
tested at the moment, but all the databases below should be easy to add
with a [contribution](#contributions)

- [x] Postgres
- [ ] MySQL (open a Pull Request)
- [ ] SQLite (open a Pull Request)
- [ ] SQL Server (open a Pull Request)

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

The `NewMigrator()` function accepts option arguments to customize the dialect
and the name of the migration tracking table. By default, the tracking table
will be set to `schema.DefaultTableName` (`schema_migrations`). To change it
to `my_migrations`:

```go
migrator := schema.NewMigrator(WithTableName("my_migrations"))
```

It is theoretically possible to create multiple `Migrator`s and to use mutliple
migration tracking tables within the same application and database.

It is also OK for multiple processes to run `Apply` on identically configured
migrators simultaneously. The `Migrator` only creates the tracking table if it
does not exist, and then locks it to modifications while building and running
the migration plan. This means that the first-arriving process will **win** and
will perform its migrations on the database.

## Rules of Applying Migrations

1. **Never, ever change** the `ID` or `Script` of a Migration which has already
been executed on your database. If you've made a mistake, you'll need to correct
it in a subsequent migration.
2. Use a consistent, but descriptive format for migration `ID`s. Your format
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
