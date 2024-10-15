package models

import (
	"context"
	"database/sql"

	_ "embed"

	permissions_models "github.com/Nigel2392/go-django/src/contrib/auth/auth-permissions/permissions-models"
	dj_models "github.com/Nigel2392/go-django/src/models"
	"github.com/mattn/go-sqlite3"
)

var _ permissions_models.Querier = (*Queries)(nil)

//go:embed schema.sqlite.sql
var sqlite_schema string

func init() {
	permissions_models.Register(
		sqlite3.SQLiteDriver{}, &dj_models.BaseBackend[permissions_models.Querier]{
			CreateTableQuery: sqlite_schema,
			NewQuerier: func(d *sql.DB) (permissions_models.Querier, error) {
				return New(d), nil
			},
			PreparedQuerier: func(ctx context.Context, d *sql.DB) (permissions_models.Querier, error) {
				return New(d), nil
			},
		},
	)
}

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

func New(db DBTX) *Queries {
	return &Queries{db: db}
}

type Queries struct {
	db DBTX
}

func (q *Queries) WithTx(tx *sql.Tx) permissions_models.Querier {
	return &Queries{
		db: tx,
	}
}

func (q *Queries) Close() error {
	return nil
}
