package models_sqlite

import (
	"context"

	_ "embed"

	"github.com/Nigel2392/go-django-queries/src/drivers"
	models "github.com/Nigel2392/go-django/src/contrib/auth/auth-models"
	dj_models "github.com/Nigel2392/go-django/src/models"
	"github.com/mattn/go-sqlite3"
)

//go:embed schema.sqlite3.sql
var sqlite_schema string

func init() {
	models.Register(
		sqlite3.SQLiteDriver{}, &dj_models.BaseBackend[models.Querier]{
			CreateTableQuery: sqlite_schema,
			NewQuerier: func(d drivers.Database) (models.Querier, error) {
				return New(d), nil
			},
			PreparedQuerier: func(ctx context.Context, d drivers.Database) (models.Querier, error) {
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

func (q *Queries) WithTx(tx drivers.Transaction) models.Querier {
	return &Queries{
		db: tx,
	}
}
