package revisions_sqlite

import (
	"context"
	"database/sql"

	"github.com/Nigel2392/go-django-queries/src/drivers"
	"github.com/Nigel2392/go-django/src/contrib/revisions/internal/revisions_db"
	"github.com/Nigel2392/go-django/src/models"
	"github.com/mattn/go-sqlite3"

	_ "embed"
)

//go:embed revisions.schema.sql
var sqlite_schema string

func init() {
	revisions_db.Register(
		sqlite3.SQLiteDriver{}, &models.BaseBackend[revisions_db.Querier]{
			CreateTableQuery: sqlite_schema,
			NewQuerier: func(d drivers.Database) (revisions_db.Querier, error) {
				return New(d), nil
			},
			PreparedQuerier: func(ctx context.Context, d drivers.Database) (revisions_db.Querier, error) {
				return New(d), nil
			},
		},
	)
}

func New(db drivers.Database) *Queries {
	return &Queries{db: db}
}

func (q *Queries) Close() error {
	return nil
}

func (q *Queries) exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return q.db.ExecContext(ctx, query, args...)
}

func (q *Queries) query(ctx context.Context, query string, args ...interface{}) (drivers.SQLRows, error) {
	return q.db.QueryContext(ctx, query, args...)
}

func (q *Queries) queryRow(ctx context.Context, query string, args ...interface{}) drivers.SQLRow {
	return q.db.QueryRowContext(ctx, query, args...)
}

type Queries struct {
	db drivers.DB
}

func (q *Queries) WithTx(tx drivers.Transaction) revisions_db.Querier {
	return &Queries{
		db: tx,
	}
}
