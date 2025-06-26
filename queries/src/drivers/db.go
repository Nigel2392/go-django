package drivers

import (
	"context"
	"database/sql"
	"database/sql/driver"
)

type SQLRow interface {
	Err() error
	Scan(dest ...any) error
}

type SQLRows interface {
	SQLRow
	Close() error
	Next() bool
	Columns() ([]string, error)
	NextResultSet() bool
}

// This interface is compatible with `*sql.DB` and `*sql.Tx`.
//
// It is used for simple transaction management in the queryset.
//
// If a transaction was started, the queryset should return the transaction instead of the database connection
// when calling [github.com/Nigel2392/go-django/queries/src.QuerySet.DB].
type DB interface {
	QueryContext(ctx context.Context, query string, args ...any) (SQLRows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) SQLRow
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type Database interface {
	DB
	Ping() error
	Driver() driver.Driver
	Begin(ctx context.Context) (Transaction, error)
	Close() error
}

// This interface is compatible with `*sql.Tx`.
//
// It is used for simple transaction management in the queryset.
type Transaction interface {
	DB
	Commit() error
	Rollback() error
}

type sqlDB[RowsT SQLRows, RowT SQLRow, ResultT any] interface {
	QueryContext(ctx context.Context, query string, args ...any) (RowsT, error)
	QueryRowContext(ctx context.Context, query string, args ...any) RowT
	ExecContext(ctx context.Context, query string, args ...any) (ResultT, error)
}

type sqlTx[RowsT SQLRows, RowT SQLRow, ResultT any] interface {
	sqlDB[RowsT, RowT, ResultT]
	Commit() error
	Rollback() error
}
