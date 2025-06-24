package models

import (
	"context"
	"database/sql"

	_ "embed"

	"github.com/Nigel2392/go-django-queries/src/drivers"
	permissions_models "github.com/Nigel2392/go-django/src/contrib/auth/auth-permissions/permissions-models"
	dj_models "github.com/Nigel2392/go-django/src/models"
	"github.com/go-sql-driver/mysql"
)

var _ permissions_models.Querier = (*Queries)(nil)

//go:embed schema.mysql.sql
var mysql_schema string

func init() {
	permissions_models.Register(
		mysql.MySQLDriver{}, &dj_models.BaseBackend[permissions_models.Querier]{
			CreateTableQuery: mysql_schema,
			NewQuerier: func(d drivers.Database) (permissions_models.Querier, error) {
				return New(d), nil
			},
			PreparedQuerier: func(ctx context.Context, d drivers.Database) (permissions_models.Querier, error) {
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

func (q *Queries) WithTx(tx drivers.Transaction) permissions_models.Querier {
	return &Queries{
		db: tx,
	}
}
