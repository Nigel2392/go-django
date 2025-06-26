package drivers

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"

	"github.com/Nigel2392/go-django/queries/src/query_errors"
)

type stdlibQuerier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

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

	return &dbWrapper{queryWrapper: queryWrapper[*sql.DB]{conn: db}}, nil
}

func SQLDBOption(opt func(driverName string, db *sql.DB) error) OpenOption {
	return func(driverName string, db any) error {
		if sqlDB, ok := db.(*sql.DB); ok {
			return opt(driverName, sqlDB)
		}
		return query_errors.ErrNotImplemented
	}
}

type queryWrapper[T stdlibQuerier] struct {
	conn T
}

func (d *queryWrapper[T]) QueryContext(ctx context.Context, query string, args ...any) (SQLRows, error) {
	var res, err = d.conn.QueryContext(ctx, query, args...)
	LogSQL(ctx, "sql.DB", err, query, args...)
	return res, err
}

func (d *queryWrapper[T]) QueryRowContext(ctx context.Context, query string, args ...any) SQLRow {
	var res = d.conn.QueryRowContext(ctx, query, args...)
	LogSQL(ctx, "sql.DB", res.Err(), query, args...)
	return res
}

func (d *queryWrapper[T]) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	var res, err = d.conn.ExecContext(ctx, query, args...)
	LogSQL(ctx, "sql.DB", err, query, args...)
	return res, err
}

type dbWrapper struct {
	queryWrapper[*sql.DB]
}

func (d *dbWrapper) Begin(ctx context.Context) (Transaction, error) {
	tx, err := d.queryWrapper.conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &txWrapper{
		queryWrapper: queryWrapper[*sql.Tx]{conn: tx},
	}, nil
}

func (d *dbWrapper) Ping(ctx context.Context) error {
	LogSQL(ctx, "sql.DB", nil, "PING")
	return d.queryWrapper.conn.PingContext(ctx)
}

func (d *dbWrapper) Driver() driver.Driver {
	return d.queryWrapper.conn.Driver()
}

func (d *dbWrapper) Close() error {
	return d.queryWrapper.conn.Close()
}

type txWrapper struct {
	queryWrapper[*sql.Tx]
}

func (t *txWrapper) Commit(ctx context.Context) error {
	var err = t.queryWrapper.conn.Commit()
	LogSQL(ctx, "sql.Tx", err, "COMMIT")
	return err
}

func (t *txWrapper) Rollback(ctx context.Context) error {
	var err = t.queryWrapper.conn.Rollback()
	LogSQL(ctx, "sql.Tx", err, "ROLLBACK")
	return err
}
