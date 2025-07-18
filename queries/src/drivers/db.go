package drivers

import (
	"context"
	"database/sql"
	"database/sql/driver"
)

type Unwrapper interface {
	// Unwrap returns the underlying database connection.
	Unwrap() any
}

// SQLRow interface represents a single row result from a database query.
//
// It provides methods to check for errors and scan the row's data into destination variables.
//
// SQLRow is typically returned by methods like QueryRowContext, allowing you to retrieve a single row of data.
type SQLRow interface {
	Err() error
	Scan(dest ...any) error
}

// SQLRows interface represents a set of rows returned by a database query.
//
// It provides methods to iterate over the rows, retrieve column names, and manage the result set.
//
// It extends the SQLRow interface to include methods for iterating through multiple rows and managing the result set.
//
// SQLRows is typically returned by methods like QueryContext, allowing you to process multiple rows of data.
type SQLRows interface {
	SQLRow
	Close() error
	Next() bool
	Columns() ([]string, error)
	NextResultSet() bool
}

// This interface is the base for all database interactions.
//
// It is used to specify a unified way to interact with the database,
// regardless of the underlying driver or database type.
//
// If a transaction was started, the queryset should return the transaction instead of the database connection
// when calling [github.com/Nigel2392/go-django/queries/src.QuerySet.DB].
type DB interface {
	Unwrap() any
	QueryContext(ctx context.Context, query string, args ...any) (SQLRows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) SQLRow
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// Database interface represents a database connection.
//
// It is used to interact with the database, execute queries, and manage transactions.
//
// An implementation of this interface can be found in `db_impl_sql.go` for SQL databases,
// or for Postgress it is in `db_impl_pgx.go`.
type Database interface {
	DB
	Ping(context.Context) error
	Driver() driver.Driver
	Begin(ctx context.Context) (Transaction, error)
	Close() error
}

// Transaction interface represents a database transaction.
//
// It extends the DB interface to include transaction management methods such as Commit and Rollback.
//
// A transaction is a sequence of operations performed in a way that ensures data integrity and consistency.
// It allows multiple operations to be executed as a single unit of work, which can be committed or rolled back as a whole,
// depending on whether all operations succeed or if any operation fails.
type Transaction interface {
	DB

	// Finished returns true if the transaction has been committed or rolled back.
	Finished() bool
	Commit(context.Context) error
	Rollback(context.Context) error
}
