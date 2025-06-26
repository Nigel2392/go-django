package drivers

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"

	"github.com/Nigel2392/go-django/queries/src/query_errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgxQuerier interface {
	Query(ctx context.Context, query string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, query string, args ...any) pgx.Row
	Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error)
	SendBatch(ctx context.Context, batch *pgx.Batch) pgx.BatchResults
}

type pgxConn interface {
	pgxQuerier
	Ping(ctx context.Context) error
	Begin(ctx context.Context) (pgx.Tx, error)
}

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

	var qW = queryWrapperPGX[*pgxpool.Pool]{conn: pool}
	var cW = connWrapperPGX[*pgxpool.Pool]{queryWrapperPGX: qW}
	return &poolWrapperPGX{connWrapperPGX: cW}, nil
}

func PGXOption(opt func(driverName string, db *pgxpool.Pool) error) OpenOption {
	return func(driverName string, db any) error {
		if pool, ok := db.(*pgxpool.Pool); ok {
			return opt(driverName, pool)
		}
		return query_errors.ErrNotImplemented
	}
}

type queryWrapperPGX[T pgxQuerier] struct {
	conn T
}

func (c *queryWrapperPGX[T]) QueryContext(ctx context.Context, query string, args ...any) (SQLRows, error) {
	var rows, err = c.conn.Query(ctx, query, args...)
	LogSQL(ctx, fmt.Sprintf("%T", c.conn), err, query, args...)
	if err != nil {
		return nil, err
	}
	return &pgxRows{Rows: rows}, nil
}

func (c *queryWrapperPGX[T]) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	result, err := c.conn.Exec(ctx, query, args...)
	LogSQL(ctx, fmt.Sprintf("%T", c.conn), err, query, args...)
	if err != nil {
		return nil, err
	}
	return &pgResult{CommandTag: result}, nil
}

func (c *queryWrapperPGX[T]) QueryRowContext(ctx context.Context, query string, args ...any) SQLRow {
	var row = c.conn.QueryRow(ctx, query, args...)
	if canErr, ok := row.(interface{ Err() error }); ok {
		LogSQL(ctx, fmt.Sprintf("%T", c.conn), canErr.Err(), query, args...)
	} else {
		LogSQL(ctx, fmt.Sprintf("%T", c.conn), nil, query, args...)
	}
	return &pgxRow{Row: row}
}

func (c *queryWrapperPGX[T]) SendBatch(ctx context.Context, batch *pgx.Batch) pgx.BatchResults {
	var sb strings.Builder
	var args []any
	for i, item := range batch.QueuedQueries {
		if i > 0 {
			sb.WriteString("; ")
		}
		sb.WriteString(item.SQL)
		args = append(args, item.Arguments...)
	}
	LogSQL(ctx, fmt.Sprintf("%T.SendBatch", c.conn), nil, sb.String(), args...)
	return c.conn.SendBatch(ctx, batch)
}

type connWrapperPGX[T pgxConn] struct {
	queryWrapperPGX[T]
}

func (c *connWrapperPGX[T]) Close() error {
	return nil
}

func (c *connWrapperPGX[T]) Ping(ctx context.Context) error {
	return c.conn.Ping(ctx)
}

func (c *connWrapperPGX[T]) Driver() driver.Driver {
	return &DriverPostgres{}
}

func (c *connWrapperPGX[T]) Begin(ctx context.Context) (Transaction, error) {
	tx, err := c.conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &pgxTx{
		queryWrapperPGX: queryWrapperPGX[pgx.Tx]{conn: tx},
		ctx:             ctx,
	}, nil
}

type poolWrapperPGX struct {
	connWrapperPGX[*pgxpool.Pool]
}

func (p *poolWrapperPGX) Close() error {
	if p.conn == nil {
		return nil
	}
	p.conn.Close()
	return nil
}

func (p *poolWrapperPGX) Acquire(ctx context.Context) (Database, error) {
	conn, err := p.conn.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	return &connWrapperPGX[*pgxpool.Conn]{queryWrapperPGX: queryWrapperPGX[*pgxpool.Conn]{conn: conn}}, nil
}

type pgxTx struct {
	queryWrapperPGX[pgx.Tx]
	ctx context.Context
}

func (p *pgxTx) Commit(ctx context.Context) error {
	return p.conn.Commit(p.ctx)
}

func (p *pgxTx) Rollback(ctx context.Context) error {
	return p.conn.Rollback(p.ctx)
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
