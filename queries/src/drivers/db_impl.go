package drivers

import (
	"context"
	"database/sql"
)

type wrappedTx[T sqlTx[RowsT, RowT, ResultT], RowsT SQLRows, RowT SQLRow, ResultT sql.Result] struct {
	db T
}

func (w *wrappedTx[T, RowsT, RowT, ResultT]) Commit() error {
	return w.db.Commit()
}

func (w *wrappedTx[T, RowsT, RowT, ResultT]) Rollback() error {
	return w.db.Rollback()
}

func (w *wrappedTx[T, RowsT, RowT, ResultT]) QueryContext(ctx context.Context, query string, args ...any) (SQLRows, error) {
	var res, err = w.db.QueryContext(ctx, query, args...)
	LogSQL(ctx, "Wrapped", err, query, args...)
	return res, err
}

func (w *wrappedTx[T, RowsT, RowT, ResultT]) QueryRowContext(ctx context.Context, query string, args ...any) SQLRow {
	var res = w.db.QueryRowContext(ctx, query, args...)
	LogSQL(ctx, "Wrapped", res.Err(), query, args...)
	return res
}

func (w *wrappedTx[T, RowsT, RowT, ResultT]) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	var res, err = w.db.ExecContext(ctx, query, args...)
	LogSQL(ctx, "Wrapped", err, query, args...)
	return res, err
}
