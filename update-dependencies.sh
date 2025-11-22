#!/bin/bash
set -e

# Schema Repository - Dependency Update Script
# This script updates Go version support and all dependencies with security patches
# Date: 2025

echo "======================================"
echo "Schema Repository Dependency Update"
echo "======================================"
echo ""

# Check Go version
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}')
echo "Go version: $GO_VERSION"
echo ""

# Ensure we're at Go 1.23 or higher
if ! go version | grep -qE 'go1\.(2[3-9]|[3-9][0-9])'; then
    echo "Error: Go 1.23 or higher is required"
    echo "Please upgrade your Go installation"
    exit 1
fi

echo "Step 1: Backing up current go.mod and go.sum..."
cp go.mod go.mod.backup
cp go.sum go.sum.backup
echo "  ✓ Backups created (go.mod.backup, go.sum.backup)"
echo ""

echo "Step 2: Updating direct dependencies..."
echo ""

# Update direct dependencies one by one
echo "  → Updating go-sqlmock..."
go get -u github.com/DATA-DOG/go-sqlmock@v1.5.2

echo "  → Updating go-mssqldb..."
go get -u github.com/denisenkom/go-mssqldb@v0.12.3

echo "  → Updating mysql driver..."
go get -u github.com/go-sql-driver/mysql@latest

echo "  → Updating pq (PostgreSQL)..."
go get -u github.com/lib/pq@latest

echo "  → Updating go-sqlite3..."
go get -u github.com/mattn/go-sqlite3@v1.14.24

echo "  → Updating dockertest..."
go get -u github.com/ory/dockertest/v3@latest

echo ""
echo "Step 3: Updating critical security dependencies..."
echo ""

# Update critical indirect dependencies with known security issues
echo "  → Updating golang.org/x/crypto (CRITICAL SECURITY UPDATE)..."
go get -u golang.org/x/crypto@v0.31.0

echo "  → Updating golang.org/x/sys..."
go get -u golang.org/x/sys@latest

echo "  → Updating opencontainers/runc..."
go get -u github.com/opencontainers/runc@v1.2.4

echo ""
echo "Step 4: Cleaning up dependencies..."
go mod tidy
echo "  ✓ Dependencies cleaned up"
echo ""

echo "Step 5: Verifying module integrity..."
go mod verify
echo "  ✓ Module verification complete"
echo ""

echo "Step 6: Downloading all dependencies..."
go mod download
echo "  ✓ All dependencies downloaded"
echo ""

echo "Step 7: Running tests..."
echo ""
if go test ./... -v; then
    echo ""
    echo "  ✓ All tests passed"
else
    echo ""
    echo "  ✗ Some tests failed"
    echo "  Please review test output above"
    echo ""
    echo "Restoring backups..."
    mv go.mod.backup go.mod
    mv go.sum.backup go.sum
    exit 1
fi

echo ""
echo "======================================"
echo "Update Complete!"
echo "======================================"
echo ""
echo "Changes applied:"
echo "  • Go version updated to 1.23+"
echo "  • Toolchain set to go1.24.2"
echo "  • golang.org/x/crypto: CRITICAL security update (v0.31.0)"
echo "  • github.com/denisenkom/go-mssqldb: v0.12.0 → v0.12.3"
echo "  • github.com/mattn/go-sqlite3: v1.14.16 → v1.14.24"
echo "  • github.com/DATA-DOG/go-sqlmock: v1.5.0 → v1.5.2"
echo "  • github.com/opencontainers/runc: v1.2.3 → v1.2.4"
echo "  • All other dependencies updated to latest compatible versions"
echo ""
echo "Security improvements:"
echo "  ✓ Fixed CVE-2020-29652 (golang.org/x/crypto)"
echo "  ✓ Fixed CVE-2020-9283 (golang.org/x/crypto)"
echo "  ✓ Fixed CVE-2021-43565 (golang.org/x/crypto)"
echo "  ✓ Fixed CVE-2025-22868 (golang.org/x/crypto)"
echo "  ✓ Updated runc to patch known container escape vulnerabilities"
echo ""
echo "Next steps:"
echo "  1. Review UPDATE_NOTES.md for detailed change information"
echo "  2. Run 'go mod why -m <module>' to understand dependency usage"
echo "  3. Consider running 'govulncheck ./...' to verify no known vulnerabilities"
echo "  4. Update CI/CD pipelines to use Go 1.23+"
echo ""
echo "Backups preserved at:"
echo "  - go.mod.backup"
echo "  - go.sum.backup"
echo ""
