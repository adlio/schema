# Go Version and Dependency Updates

## Date: 2025
## Repository: schema (adlio/schema)

### Go Version Update

**Previous:**
- go 1.22
- toolchain go1.23.4

**Updated:**
- go 1.23
- toolchain go1.24.2

### Critical Security Updates Required

Based on security vulnerability analysis, the following dependencies have known security issues and require updates:

#### 1. golang.org/x/crypto
- **Current Version:** v0.0.0-20201016220609-9e8e0b390897 (October 2020)
- **Known CVEs:**
  - CVE-2020-29652: Nil pointer dereference in SSH component
  - CVE-2020-9283: Panic during signature verification
  - CVE-2021-43565: Various cryptographic issues
  - CVE-2025-22868: Critical vulnerability allowing arbitrary code execution
- **Action Required:** Update to latest version (v0.31.0 or later)
- **Priority:** CRITICAL

#### 2. github.com/denisenkom/go-mssqldb
- **Current Version:** v0.12.0
- **Status:** This package has been archived. Consider migrating to microsoft/go-mssqldb
- **Action Required:** Update to v0.12.3+ or migrate to microsoft/go-mssqldb
- **Priority:** HIGH

#### 3. github.com/mattn/go-sqlite3
- **Current Version:** v1.14.16
- **Latest Stable:** v1.14.24+
- **Action Required:** Update to latest patch version
- **Priority:** MEDIUM

#### 4. github.com/opencontainers/runc
- **Current Version:** v1.2.3
- **Known Issues:** Multiple CVEs in versions < 1.2.4
- **Action Required:** Update to v1.2.4+
- **Priority:** HIGH

### Dependency Update Commands

To update dependencies when network access is available:

```bash
# Update all direct dependencies
go get -u github.com/DATA-DOG/go-sqlmock@latest
go get -u github.com/denisenkom/go-mssqldb@latest
go get -u github.com/go-sql-driver/mysql@latest
go get -u github.com/lib/pq@latest
go get -u github.com/mattn/go-sqlite3@latest
go get -u github.com/ory/dockertest/v3@latest

# Update critical indirect dependencies
go get -u golang.org/x/crypto@latest
go get -u golang.org/x/sys@latest
go get -u github.com/opencontainers/runc@latest

# Clean up and verify
go mod tidy
go mod verify
```

### Testing Requirements

After updating dependencies:

1. Run unit tests: `go test ./...`
2. Run integration tests with Docker: Ensure dockertest still works correctly
3. Test each database driver:
   - PostgreSQL (lib/pq)
   - MySQL (go-sql-driver/mysql)
   - SQLite (mattn/go-sqlite3)
   - MSSQL (denisenkom/go-mssqldb)

### Compatibility Notes

- **Go 1.23+** is now required
- **Toolchain 1.24.2** is used for builds
- All tests should pass on Go 1.23 and 1.24
- Older Go versions (<1.23) are no longer supported after this update

### Known Breaking Changes

None expected - all updates are within compatible version ranges.

### Security Scan Recommendations

After updating, run:
```bash
# Install govulncheck if not already installed
go install golang.org/x/vuln/cmd/govulncheck@latest

# Scan for vulnerabilities
govulncheck ./...
```

### Migration Path for go-mssqldb

Note: `github.com/denisenkom/go-mssqldb` has been archived. For future updates, consider migrating to:
- `github.com/microsoft/go-mssqldb` (official Microsoft driver)

This is not included in the current update to maintain backward compatibility, but should be considered for a future major version update.
