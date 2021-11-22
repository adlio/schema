package schema

import (
	"database/sql"

	// Postgres database driver
	_ "github.com/lib/pq"
)

// Interface verification that *sql.DB satisfies our Connection interface
var (
	_ Connection = &sql.DB{}
)

// Interface verification that *sql.DB satisfies our Transactor interface
var (
	_ Transactor = &sql.DB{}
)

// Interface verification that *sql.Tx and *sql.DB both satisfy our
// Queryer interface
var (
	_ Queryer = &sql.DB{}
	_ Queryer = &sql.Tx{}
)
