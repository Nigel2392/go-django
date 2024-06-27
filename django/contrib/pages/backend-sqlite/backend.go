package models_sqlite

import (
	"context"
	"database/sql"

	"github.com/Nigel2392/django/contrib/pages/models"
	dj_models "github.com/Nigel2392/django/models"
	"github.com/mattn/go-sqlite3"

	_ "embed"
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
				return Prepare(ctx, d)
			},
		},
	)
}
