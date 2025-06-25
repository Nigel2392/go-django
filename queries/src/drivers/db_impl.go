package drivers

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"

	"github.com/Nigel2392/go-django/queries/src/query_errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

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

type wrappedDB[T sqlDB[RowsT, RowT, ResultT], RowsT SQLRows, RowT SQLRow, ResultT sql.Result] struct {
	db T
}

func (w *wrappedDB[T, RowsT, RowT, ResultT]) QueryContext(ctx context.Context, query string, args ...any) (SQLRows, error) {
	return w.db.QueryContext(ctx, query, args...)
}

func (w *wrappedDB[T, RowsT, RowT, ResultT]) QueryRowContext(ctx context.Context, query string, args ...any) SQLRow {
	return w.db.QueryRowContext(ctx, query, args...)
}

func (w *wrappedDB[T, RowsT, RowT, ResultT]) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return w.db.ExecContext(ctx, query, args...)
}

type wrappedTx[T sqlTx[RowsT, RowT, ResultT], RowsT SQLRows, RowT SQLRow, ResultT sql.Result] struct {
	wrappedDB[T, RowsT, RowT, ResultT]
}

func (w *wrappedTx[T, RowsT, RowT, ResultT]) Commit() error {
	return w.wrappedDB.db.Commit()
}

func (w *wrappedTx[T, RowsT, RowT, ResultT]) Rollback() error {
	return w.wrappedDB.db.Rollback()
}

type dbWrapper struct {
	*sql.DB
}

func (d *dbWrapper) Begin(ctx context.Context) (Transaction, error) {
	tx, err := d.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &wrappedTx[*sql.Tx, *sql.Rows, *sql.Row, sql.Result]{
		wrappedDB: wrappedDB[*sql.Tx, *sql.Rows, *sql.Row, sql.Result]{db: tx},
	}, nil
}

func (d *dbWrapper) QueryContext(ctx context.Context, query string, args ...any) (SQLRows, error) {
	return d.DB.QueryContext(ctx, query, args...)
}

func (d *dbWrapper) QueryRowContext(ctx context.Context, query string, args ...any) SQLRow {
	return d.DB.QueryRowContext(ctx, query, args...)
}

func (d *dbWrapper) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return d.DB.ExecContext(ctx, query, args...)
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

	return &dbWrapper{DB: db}, nil
}

type connWrapper struct {
	conn *pgx.Conn
}

func (c *connWrapper) Conn() *pgx.Conn {
	return c.conn
}

func (c *connWrapper) Driver() driver.Driver {
	return &DriverPostgres{}
}

func (c *connWrapper) Ping() error {
	if err := c.conn.Ping(context.Background()); err != nil {
		return err
	}
	return nil
}

func (c *connWrapper) Close() error {
	return c.conn.Close(context.Background())
}

func (c *connWrapper) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	result, err := c.conn.Exec(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &pgResult{CommandTag: result}, nil
}

func (c *connWrapper) Begin(ctx context.Context) (Transaction, error) {
	tx, err := c.conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &wrappedTx[*pgxTx, SQLRows, SQLRow, sql.Result]{
		wrappedDB: wrappedDB[*pgxTx, SQLRows, SQLRow, sql.Result]{db: &pgxTx{Tx: tx, ctx: ctx}},
	}, nil
}

func (c *connWrapper) QueryContext(ctx context.Context, query string, args ...any) (SQLRows, error) {
	var rows, err = c.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &pgxRows{Rows: rows}, nil
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
	return nil
}

func (c *connWrapper) QueryRowContext(ctx context.Context, query string, args ...any) SQLRow {
	var row = c.conn.QueryRow(ctx, query, args...)
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
	result, err := p.Exec(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &pgResult{CommandTag: result}, nil
}

func (p *pgxTx) QueryContext(ctx context.Context, query string, args ...any) (SQLRows, error) {
	rows, err := p.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &pgxRows{Rows: rows}, nil
}

func (p *pgxTx) QueryRowContext(ctx context.Context, query string, args ...any) SQLRow {
	row := p.QueryRow(ctx, query, args...)
	return &pgxRow{Row: row}
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

func OpenPGX(ctx context.Context, dsn string, opts ...OpenOption) (Database, error) {
	var conn, err = pgx.Connect(ctx, dsn)
	if err != nil {
		return nil, err
	}

	for _, opt := range opts {
		if err := opt(POSTGRES_DRIVER_NAME, conn); err != nil && !errors.Is(err, query_errors.ErrNotImplemented) {
			return nil, err
		}
	}

	return &connWrapper{conn: conn}, nil
}

func SQLDBOption(opt func(driverName string, db *sql.DB) error) OpenOption {
	return func(driverName string, db any) error {
		if sqlDB, ok := db.(*sql.DB); ok {
			return opt(driverName, sqlDB)
		}
		return query_errors.ErrNotImplemented
	}
}

func PGXOption(opt func(driverName string, db *pgx.Conn) error) OpenOption {
	return func(driverName string, db any) error {
		if pgxConn, ok := db.(*pgx.Conn); ok {
			return opt(driverName, pgxConn)
		}
		return query_errors.ErrNotImplemented
	}
}
