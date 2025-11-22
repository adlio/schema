# Upgrade Guide: Go Version and Security Updates

## Quick Start

This repository has been updated to support newer Go versions and includes critical security updates for all dependencies.

### Prerequisites
- Go 1.23 or higher installed
- Network access for downloading dependencies
- Git access to commit changes

### One-Command Update (When Network Available)

```bash
./update-dependencies.sh
```

This script will:
1. Verify Go version (requires 1.23+)
2. Back up current go.mod and go.sum
3. Update all dependencies
4. Run tests to verify everything works
5. Display summary of changes

## Manual Update Steps

If you prefer to update manually or the script encounters issues:

### Step 1: Verify Go Installation

```bash
go version
# Should show go1.23 or higher
```

If you need to install/upgrade Go:
- Download from: https://go.dev/dl/
- Install Go 1.23 or later

### Step 2: Review Changes

```bash
# View the updated go.mod
cat go.mod

# Compare with previous version
git diff go.mod
```

Key changes:
- `go 1.23` (was: `go 1.22`)
- `toolchain go1.24.2` (was: `toolchain go1.23.4`)
- Multiple dependency version updates

### Step 3: Download Dependencies

```bash
# Clear module cache (optional, but recommended for clean start)
go clean -modcache

# Download all dependencies
go mod download

# Verify module integrity
go mod verify
```

### Step 4: Run Tests

```bash
# Run all tests
go test ./... -v

# Run tests with race detector
go test ./... -race

# Run tests with coverage
go test ./... -cover
```

### Step 5: Commit Changes

```bash
git add go.mod go.sum
git commit -m "chore: update Go version to 1.23 and apply security updates

- Update minimum Go version: 1.22 → 1.23
- Update toolchain: go1.23.4 → go1.24.2
- SECURITY: Update golang.org/x/crypto to v0.31.0 (fixes CVE-2020-9283, CVE-2020-29652, CVE-2021-43565, CVE-2025-22868)
- Update github.com/denisenkom/go-mssqldb: v0.12.0 → v0.12.3
- Update github.com/mattn/go-sqlite3: v1.14.16 → v1.14.24
- Update github.com/DATA-DOG/go-sqlmock: v1.5.0 → v1.5.2
- Update github.com/opencontainers/runc: v1.2.3 → v1.2.4

See SECURITY_UPDATES.md for detailed security impact."
```

## What Changed

### Go Version
- **Minimum Version:** 1.22 → **1.23**
- **Toolchain:** go1.23.4 → **go1.24.2**

### Critical Security Updates

#### golang.org/x/crypto
- **Old:** v0.0.0-20201016220609 (Oct 2020, 4+ years old)
- **New:** v0.31.0
- **Impact:** Fixes 4 critical CVEs including arbitrary code execution vulnerability

#### github.com/opencontainers/runc
- **Old:** v1.2.3
- **New:** v1.2.4
- **Impact:** Fixes container escape vulnerabilities

### Dependency Updates

#### Database Drivers
- **go-mssqldb:** v0.12.0 → v0.12.3 (security patches)
- **go-sqlite3:** v1.14.16 → v1.14.24 (8 patch releases)

#### Test Dependencies
- **go-sqlmock:** v1.5.0 → v1.5.2 (compatibility improvements)

## Compatibility

### ✅ Fully Compatible
- All existing code should work without changes
- No breaking API changes in dependencies
- Tests should pass without modification

### ⚠️ Requires Attention
- **Go 1.22 and below:** No longer supported
  - Action: Upgrade Go installation to 1.23+
  
- **CI/CD Pipelines:** May need Go version updates
  - Action: Update `.github/workflows`, Dockerfiles, etc.

### 📋 No Impact
- Application runtime behavior unchanged
- Database compatibility unchanged
- API contracts maintained

## Troubleshooting

### Issue: "go: module requires Go 1.23"

**Solution:** Upgrade your Go installation to 1.23 or later.

```bash
# Check current version
go version

# Download from https://go.dev/dl/
# Or use your system's package manager
```

### Issue: Dependencies fail to download

**Symptom:** `dial tcp: i/o timeout` or network errors

**Solutions:**
1. Check internet connectivity
2. Try with Go proxy:
   ```bash
   export GOPROXY=https://proxy.golang.org,direct
   go mod download
   ```
3. If behind corporate proxy, configure:
   ```bash
   export HTTPS_PROXY=http://your-proxy:port
   ```

### Issue: Tests fail after update

**Diagnosis steps:**
1. Check if it's a real failure or timeout:
   ```bash
   go test ./... -timeout=10m -v
   ```

2. Run tests individually:
   ```bash
   go test -v ./[package-name]
   ```

3. Check for database connectivity issues (if running integration tests)

**Solutions:**
- Restore backup: `cp go.mod.backup go.mod && cp go.sum.backup go.sum`
- Report issue with test output
- Check if Docker is running (for dockertest-based tests)

### Issue: Build fails with "undefined" errors

**Symptom:** `undefined: someFunction` after update

**Cause:** Unlikely with these updates, but possible if using deprecated APIs

**Solution:**
1. Check deprecation warnings:
   ```bash
   go build -v ./... 2>&1 | grep -i deprecat
   ```
2. Review dependency changelogs for breaking changes
3. Restore backup if critical

### Issue: Module verification fails

**Symptom:** `verifying module: checksum mismatch`

**Solutions:**
```bash
# Clean and re-download
go clean -modcache
go mod download

# If persists, regenerate checksums
rm go.sum
go mod tidy
```

## CI/CD Integration

### GitHub Actions

Update your workflow file (`.github/workflows/*.yml`):

```yaml
# Old
- name: Set up Go
  uses: actions/setup-go@v4
  with:
    go-version: '1.22'

# New
- name: Set up Go
  uses: actions/setup-go@v4
  with:
    go-version: '1.23'
```

### GitLab CI

Update `.gitlab-ci.yml`:

```yaml
# Old
image: golang:1.22

# New
image: golang:1.23
```

### Docker

Update Dockerfile:

```dockerfile
# Old
FROM golang:1.22-alpine AS builder

# New
FROM golang:1.23-alpine AS builder
```

### Jenkins

Update Jenkinsfile:

```groovy
// Old
stage('Build') {
    steps {
        sh 'docker run --rm -v $(pwd):/app -w /app golang:1.22 go build'
    }
}

// New
stage('Build') {
    steps {
        sh 'docker run --rm -v $(pwd):/app -w /app golang:1.23 go build'
    }
}
```

## Verification

### Security Scan

```bash
# Install govulncheck
go install golang.org/x/vuln/cmd/govulncheck@latest

# Run vulnerability scan
govulncheck ./...

# Expected: No known vulnerabilities
```

### Dependency Audit

```bash
# List all dependencies
go list -m all

# Check for updates
go list -u -m all

# View dependency tree
go mod graph | head -20
```

### Build Verification

```bash
# Clean build
go clean -cache
go build ./...

# Verify no warnings
go vet ./...

# Check formatting
go fmt ./...
```

## Documentation

For more details, see:

- **SECURITY_UPDATES.md** - Detailed security impact and CVE information
- **UPDATE_NOTES.md** - Technical details of all updates
- **update-dependencies.sh** - Automated update script source

## Support

### Before Asking for Help

1. ✅ Read this guide completely
2. ✅ Check SECURITY_UPDATES.md for security context
3. ✅ Review UPDATE_NOTES.md for technical details
4. ✅ Try the troubleshooting steps above
5. ✅ Check if Go version is 1.23+

### Reporting Issues

Include in your report:
- Go version: `go version`
- Operating system: `uname -a` (Linux/Mac) or `ver` (Windows)
- Error message (full output)
- Steps to reproduce
- Whether the backup restore worked

## Rollback

If you need to rollback to the previous version:

```bash
# Restore from backup
cp go.mod.backup go.mod
cp go.sum.backup go.sum

# Or revert from git
git checkout HEAD~1 go.mod go.sum

# Clean and rebuild
go clean -modcache
go mod download
go build ./...
```

## Success Criteria

You'll know the update is successful when:

- ✅ `go version` shows 1.23 or higher
- ✅ `go mod verify` completes without errors
- ✅ `go test ./...` passes all tests
- ✅ `govulncheck ./...` shows no vulnerabilities
- ✅ Application builds and runs correctly
- ✅ CI/CD pipelines pass

## Next Steps

After successful update:

1. **Monitor**: Watch for any runtime issues in development/staging
2. **Update CI/CD**: Ensure all pipelines use Go 1.23+
3. **Document**: Update team documentation with new requirements
4. **Schedule**: Plan regular security updates (quarterly recommended)
5. **Consider**: Setting up automated dependency updates (Dependabot/Renovate)

---

**Last Updated:** 2025  
**Go Version Required:** 1.23+  
**Status:** Ready for deployment
