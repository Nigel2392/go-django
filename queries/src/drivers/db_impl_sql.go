package drivers

import (
	"context"
	"database/sql"
	"database/sql/driver"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
)

type stdlibQuerier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func OpenSQL(driverName string, drv *Driver, dsn string, opts ...OpenOption) (Database, error) {
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, err
	}

	for _, opt := range opts {
		if err := opt(driverName, db); err != nil && !errors.Is(err, errors.NotImplemented) {
			return nil, err
		}
	}

	return &dbWrapper{queryWrapper: queryWrapper[*sql.DB]{conn: db, d: drv}}, nil
}

func SQLDBOption(opt func(driverName string, db *sql.DB) error) OpenOption {
	return func(driverName string, db any) error {
		if sqlDB, ok := db.(*sql.DB); ok {
			return opt(driverName, sqlDB)
		}
		return errors.NotImplemented
	}
}

type queryWrapper[T stdlibQuerier] struct {
	conn T
	d    *Driver
}

func (d *queryWrapper[T]) Unwrap() any {
	if unwrapper, ok := any(d.conn).(Unwrapper); ok {
		return unwrapper.Unwrap()
	}
	return d.conn
}

func (d *queryWrapper[T]) QueryContext(ctx context.Context, query string, args ...any) (SQLRows, error) {
	var res, err = d.conn.QueryContext(ctx, query, args...)
	LogSQL(ctx, "sql.DB", err, query, args...)
	return &sqlRowsWrapper{Rows: res, d: d.d}, databaseError(d.d, err)
}

func (d *queryWrapper[T]) QueryRowContext(ctx context.Context, query string, args ...any) SQLRow {
	var res = d.conn.QueryRowContext(ctx, query, args...)
	LogSQL(ctx, "sql.DB", res.Err(), query, args...)
	return &sqlRowWrapper{
		Row: res,
		d:   d.d,
	}
}

func (d *queryWrapper[T]) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	var res, err = d.conn.ExecContext(ctx, query, args...)
	LogSQL(ctx, "sql.DB", err, query, args...)
	return res, databaseError(d.d, err)
}

type dbWrapper struct {
	queryWrapper[*sql.DB]
}

func (d *dbWrapper) Begin(ctx context.Context) (Transaction, error) {
	var tx, err = d.queryWrapper.conn.BeginTx(ctx, nil)
	LogSQL(ctx, "sql.DB", err, "BEGIN")
	if err != nil {
		return nil, databaseError(d.d, err)
	}
	return &txWrapper{
		queryWrapper: queryWrapper[*sql.Tx]{conn: tx, d: d.d},
	}, nil
}

func (d *dbWrapper) Ping(ctx context.Context) error {
	LogSQL(ctx, "sql.DB", nil, "PING")
	return databaseError(d.d, d.queryWrapper.conn.PingContext(ctx))
}

func (d *dbWrapper) Driver() driver.Driver {
	return d.queryWrapper.conn.Driver()
}

func (d *dbWrapper) Close() error {
	return databaseError(d.d, d.queryWrapper.conn.Close())
}

type txWrapper struct {
	queryWrapper[*sql.Tx]
	finished bool
}

func (p *txWrapper) Finished() bool {
	return p.finished
}

func (t *txWrapper) Commit(ctx context.Context) error {
	defer func() { t.finished = true }()
	var err = t.queryWrapper.conn.Commit()
	LogSQL(ctx, "sql.Tx", err, "COMMIT")
	return databaseError(t.d, err)
}

func (t *txWrapper) Rollback(ctx context.Context) error {
	defer func() { t.finished = true }()
	var err = t.queryWrapper.conn.Rollback()
	LogSQL(ctx, "sql.Tx", err, "ROLLBACK")
	return databaseError(t.d, err)
}

type sqlRowWrapper struct {
	*sql.Row
	d *Driver
}

func (r *sqlRowWrapper) Err() error {
	return databaseError(r.d, r.Row.Err())
}

type sqlRowsWrapper struct {
	*sql.Rows
	d *Driver
}

func (r *sqlRowsWrapper) Scan(dest ...any) error {
	err := r.Rows.Scan(dest...)
	return databaseError(r.d, err)
}

func (r *sqlRowsWrapper) Err() error {
	err := r.Rows.Err()
	return databaseError(r.d, err)
}
