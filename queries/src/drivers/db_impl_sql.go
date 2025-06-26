package drivers

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Nigel2392/go-django/queries/src/query_errors"
)

func OpenSQL(driverName, dsn string, opts ...OpenOption) (Database, error) {
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, err
	}

	for _, opt := range opts {
		if err := opt(driverName, db); err != nil && !errors.Is(err, query_errors.ErrNotImplemented) {
			return nil, err
		}
	}

	return &dbWrapper{DB: db}, nil
}

func SQLDBOption(opt func(driverName string, db *sql.DB) error) OpenOption {
	return func(driverName string, db any) error {
		if sqlDB, ok := db.(*sql.DB); ok {
			return opt(driverName, sqlDB)
		}
		return query_errors.ErrNotImplemented
	}
}

type dbWrapper struct {
	*sql.DB
}

func (d *dbWrapper) Begin(ctx context.Context) (Transaction, error) {
	tx, err := d.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &wrappedTx[*sql.Tx, *sql.Rows, *sql.Row, sql.Result]{db: tx}, nil
}

func (d *dbWrapper) QueryContext(ctx context.Context, query string, args ...any) (SQLRows, error) {
	var res, err = d.DB.QueryContext(ctx, query, args...)
	LogSQL(ctx, "sql.DB", err, query, args...)
	return res, err
}

func (d *dbWrapper) QueryRowContext(ctx context.Context, query string, args ...any) SQLRow {
	var res = d.DB.QueryRowContext(ctx, query, args...)
	LogSQL(ctx, "sql.DB", res.Err(), query, args...)
	return res
}

func (d *dbWrapper) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	var res, err = d.DB.ExecContext(ctx, query, args...)
	LogSQL(ctx, "sql.DB", err, query, args...)
	return res, err
}
