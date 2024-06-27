package models_sqlite

import (
	"context"
	"database/sql"

	_ "embed"

	models "github.com/Nigel2392/django/contrib/auth/auth-models"
	dj_models "github.com/Nigel2392/django/models"
	"github.com/mattn/go-sqlite3"
)

//go:embed schema.sqlite3.sql
var sqlite_schema string

func init() {
	models.Register(
		sqlite3.SQLiteDriver{}, &dj_models.BaseBackend[models.Querier]{
			CreateTableQuery: sqlite_schema,
			NewQuerier: func(d *sql.DB) (models.Querier, error) {
				return New(d), nil
			},
			PreparedQuerier: func(ctx context.Context, d *sql.DB) (models.Querier, error) {
				return New(d), nil
			},
		},
	)
}

func New(db models.DBTX) *Queries {
	return &Queries{db: db}
}

type Queries struct {
	db models.DBTX
}

func (q *Queries) WithTx(tx *sql.Tx) models.Querier {
	return &Queries{
		db: tx,
	}
}
