# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.5.0] - 2026-04-18

### Changed

- **Breaking:** Go 1.25.5+ is now required because the Docker-based test stack now depends on `ory/dockertest/v4`
- Update Docker-based test dependencies to address security vulnerabilities by migrating to `ory/dockertest/v4`
- Finalize GitHub Actions CI/release automation and improve automated release notes extraction from README version history

## [1.4.0] - 2026-02-21

### Changed

- **Breaking:** Go 1.24+ now required (was 1.22)
- **Breaking:** MSSQL driver changed from `denisenkom/go-mssqldb` to `microsoft/go-mssqldb`
- Update all dependencies to latest versions
- Migrate CI from CircleCI to GitHub Actions
- Add automated release workflow

### Fixed

- Fix data race in MSSQL Unlock function

## [1.3.9] - 2025-07-21

### Changed

- Update Go version requirement to 1.21 to resolve CircleCI build issues with slices package dependency
- Update CircleCI configuration to use Go 1.21

## [1.3.8] - 2025-07-19

### Fixed

- Update golang.org/x/crypto to v0.40.0 to address security vulnerabilities
- Update golang.org/x/net to v0.42.0 to address security vulnerabilities

## [1.3.7] - 2025-07-19

### Added

- SQL Server support for the Locker interface using sp_getapplock/sp_releaseapplock

### Fixed

- Fix SQL Server transaction handling for concurrent migrations

## [1.3.4] - 2023-04-09

### Fixed

- Update downstream dependencies to address vulnerabilities in test dependencies

## [1.3.3] - 2022-06-19

### Fixed

- Update downstream dependencies of ory/dockertest due to security issues

## [1.3.0] - 2022-03-25

### Added

- Basic SQL Server support (no locking, not recommended for use in clusters)

### Changed

- Improved support for running tests on ARM64 machines (M1 Macs)

## [1.2.3] - 2021-12-10

### Fixed

- Restore the ability to chain NewMigrator().Apply

## [1.2.2] - 2021-12-09

### Added

- Support for migrations in an embed.FS (`FSMigrations(filesystem fs.FS, glob string)`)
- MySQL/MariaDB support (experimental)
- SQLite support (experimental)

### Changed

- Update go.mod to `go 1.17`

## [1.1.14] - 2021-11-18

### Fixed

- Security patches in upstream dependencies

## [1.1.13] - 2020-05-22

### Fixed

- Bugfix for error with advisory lock being held open
- Improved test coverage for simultaneous execution

## [1.1.11] - 2020-05-19

### Changed

- Use a database-held lock for all migrations not just the initial table creation

## [1.1.9] - 2020-05-17

### Added

- Add the ability to attach a logger

## [1.1.8] - 2019-11-24

### Changed

- Switch to `filepath` package for improved cross-platform filesystem support

## [1.1.7] - 2019-10-01

### Added

- Use pg_advisory_lock() to prevent race conditions when multiple processes/machines try to simultaneously create the migrations table

## [1.1.1] - 2019-09-28

### Added

- First published version

[Unreleased]: https://github.com/adlio/schema/compare/v1.5.0...HEAD
[1.5.0]: https://github.com/adlio/schema/compare/v1.4.0...v1.5.0
[1.4.0]: https://github.com/adlio/schema/compare/v1.3.9...v1.4.0
[1.3.9]: https://github.com/adlio/schema/compare/v1.3.8...v1.3.9
[1.3.8]: https://github.com/adlio/schema/compare/v1.3.7...v1.3.8
[1.3.7]: https://github.com/adlio/schema/compare/v1.3.4...v1.3.7
[1.3.4]: https://github.com/adlio/schema/compare/v1.3.3...v1.3.4
[1.3.3]: https://github.com/adlio/schema/compare/v1.3.0...v1.3.3
[1.3.0]: https://github.com/adlio/schema/compare/v1.2.3...v1.3.0
[1.2.3]: https://github.com/adlio/schema/compare/v1.2.2...v1.2.3
[1.2.2]: https://github.com/adlio/schema/compare/v1.1.14...v1.2.2
[1.1.14]: https://github.com/adlio/schema/compare/v1.1.13...v1.1.14
[1.1.13]: https://github.com/adlio/schema/compare/v1.1.11...v1.1.13
[1.1.11]: https://github.com/adlio/schema/compare/v1.1.9...v1.1.11
[1.1.9]: https://github.com/adlio/schema/compare/v1.1.8...v1.1.9
[1.1.8]: https://github.com/adlio/schema/compare/v1.1.7...v1.1.8
[1.1.7]: https://github.com/adlio/schema/compare/v1.1.1...v1.1.7
[1.1.1]: https://github.com/adlio/schema/releases/tag/v1.1.1
