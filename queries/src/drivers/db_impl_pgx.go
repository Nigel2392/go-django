package drivers

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"

	"github.com/Nigel2392/go-django/queries/src/query_errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func OpenPGX(ctx context.Context, dsn string, opts ...OpenOption) (Database, error) {
	var pool, err = pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}

	for _, opt := range opts {
		if err := opt(POSTGRES_DRIVER_NAME, pool); err != nil && !errors.Is(err, query_errors.ErrNotImplemented) {
			return nil, err
		}
	}

	return &poolWrapper{pool: pool}, nil
}

func PGXOption(opt func(driverName string, db *pgxpool.Pool) error) OpenOption {
	return func(driverName string, db any) error {
		if pool, ok := db.(*pgxpool.Pool); ok {
			return opt(driverName, pool)
		}
		return query_errors.ErrNotImplemented
	}
}

type poolWrapper struct {
	pool *pgxpool.Pool
}

func (p *poolWrapper) Ping() error {
	return p.pool.Ping(context.Background())
}

func (p *poolWrapper) Driver() driver.Driver {
	return &DriverPostgres{}
}

func (p *poolWrapper) Close() error {
	if p.pool == nil {
		return nil
	}
	p.pool.Close()
	return nil
}

func (c *poolWrapper) Begin(ctx context.Context) (Transaction, error) {
	tx, err := c.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &wrappedTx[*pgxTx, SQLRows, SQLRow, sql.Result]{db: &pgxTx{Tx: tx, ctx: ctx}}, nil
}

func (p *poolWrapper) SendBatch(ctx context.Context, batch *pgx.Batch) pgx.BatchResults {
	var sb strings.Builder
	var args []any
	for i, item := range batch.QueuedQueries {
		if i > 0 {
			sb.WriteString("; ")
		}
		sb.WriteString(item.SQL)
		args = append(args, item.Arguments...)
	}
	LogSQL(ctx, "pgxpool.Pool.SendBatch", nil, sb.String(), args...)
	return p.pool.SendBatch(ctx, batch)
}

func (p *poolWrapper) Acquire(ctx context.Context) (Database, error) {
	conn, err := p.pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	return &connWrapper{conn: conn}, nil
}

func (p *poolWrapper) QueryContext(ctx context.Context, query string, args ...any) (SQLRows, error) {
	var rows, err = p.pool.Query(ctx, query, args...)
	LogSQL(ctx, "pgxpool.Pool", err, query, args...)
	if err != nil {
		return nil, err
	}
	return &pgxRows{Rows: rows}, nil
}

func (p *poolWrapper) QueryRowContext(ctx context.Context, query string, args ...any) SQLRow {
	var row = p.pool.QueryRow(ctx, query, args...)
	if canErr, ok := row.(interface{ Err() error }); ok {
		LogSQL(ctx, "pgxpool.Pool", canErr.Err(), query, args...)
	} else {
		LogSQL(ctx, "pgxpool.Pool", nil, query, args...)
	}
	return &pgxRow{Row: row}
}

func (p *poolWrapper) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	result, err := p.pool.Exec(ctx, query, args...)
	LogSQL(ctx, "pgxpool.Pool", err, query, args...)
	if err != nil {
		return nil, err
	}
	return &pgResult{CommandTag: result}, nil
}

type connWrapper struct {
	conn *pgxpool.Conn
}

func (c *connWrapper) Close() error {
	return nil
}

func (c *connWrapper) Ping() error {
	if c.conn == nil {
		return query_errors.ErrNoDatabase
	}
	return c.conn.Ping(context.Background())
}

func (c *connWrapper) Driver() driver.Driver {
	return &DriverPostgres{}
}

func (c *connWrapper) Begin(ctx context.Context) (Transaction, error) {
	tx, err := c.conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &wrappedTx[*pgxTx, SQLRows, SQLRow, sql.Result]{db: &pgxTx{Tx: tx, ctx: ctx}}, nil
}

func (c *connWrapper) QueryContext(ctx context.Context, query string, args ...any) (SQLRows, error) {
	var rows, err = c.conn.Query(ctx, query, args...)
	LogSQL(ctx, "pgxpool.Conn", err, query, args...)
	if err != nil {
		return nil, err
	}
	return &pgxRows{Rows: rows}, nil
}

func (c *connWrapper) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	result, err := c.conn.Exec(ctx, query, args...)
	LogSQL(ctx, "pgxpool.Conn", err, query, args...)
	if err != nil {
		return nil, err
	}
	return &pgResult{CommandTag: result}, nil
}

func (c *connWrapper) QueryRowContext(ctx context.Context, query string, args ...any) SQLRow {
	var row = c.conn.QueryRow(ctx, query, args...)
	if canErr, ok := row.(interface{ Err() error }); ok {
		LogSQL(ctx, "pgxpool.Conn", canErr.Err(), query, args...)
	} else {
		LogSQL(ctx, "pgxpool.Conn", nil, query, args...)
	}
	return &pgxRow{Row: row}
}

type pgxTx struct {
	pgx.Tx
	ctx context.Context
}

func (p *pgxTx) Commit() error {
	return p.Tx.Commit(p.ctx)
}

func (p *pgxTx) Rollback() error {
	return p.Tx.Rollback(p.ctx)
}

func (p *pgxTx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	var result, err = p.Exec(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &pgResult{CommandTag: result}, nil
}

func (p *pgxTx) QueryContext(ctx context.Context, query string, args ...any) (SQLRows, error) {
	var rows, err = p.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &pgxRows{Rows: rows}, nil
}

func (p *pgxTx) QueryRowContext(ctx context.Context, query string, args ...any) SQLRow {
	var row = p.QueryRow(ctx, query, args...)
	return &pgxRow{Row: row}
}

type pgxRows struct {
	pgx.Rows
}

func (r *pgxRows) Close() error {
	r.Rows.Close()
	return nil
}

func (r *pgxRows) NextResultSet() bool {
	return false
}

func (r *pgxRows) Columns() ([]string, error) {
	columns := r.Rows.FieldDescriptions()
	result := make([]string, len(columns))
	for i, col := range columns {
		result[i] = string(col.Name)
	}
	return result, nil
}

type pgxRow struct {
	pgx.Row
}

func (r *pgxRow) Err() error {
	if canErr, ok := r.Row.(interface{ Err() error }); ok {
		return canErr.Err()
	}
	return nil
}

type pgResult struct {
	pgconn.CommandTag
}

func (p *pgResult) LastInsertId() (int64, error) {
	return 0, query_errors.ErrNotImplemented
}

func (p *pgResult) RowsAffected() (int64, error) {
	return p.CommandTag.RowsAffected(), nil
}
