package models

import (
	"context"

	_ "embed"

	"github.com/Nigel2392/go-django-queries/src/drivers"
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

type Queries struct {
	db drivers.DB
}

func (q *Queries) WithTx(tx drivers.Transaction) permissions_models.Querier {
	return &Queries{
		db: tx,
	}
}

func (q *Queries) Close() error {
	return nil
}
