package openauth2_models_sqlite

import (
	"context"
	"database/sql"

	openauth2models "github.com/Nigel2392/go-django/src/contrib/openauth2/openauth2_models"
	dj_models "github.com/Nigel2392/go-django/src/models"
	"github.com/mattn/go-sqlite3"

	_ "embed"
)

//go:embed sql/schema.sql
var sqlite_schema string

func init() {
	openauth2models.Register(
		sqlite3.SQLiteDriver{}, &dj_models.BaseBackend[openauth2models.Querier]{
			CreateTableQuery: sqlite_schema,
			NewQuerier: func(d *sql.DB) (openauth2models.Querier, error) {
				return New(d), nil
			},
			PreparedQuerier: func(ctx context.Context, d *sql.DB) (openauth2models.Querier, error) {
				return Prepare(ctx, d)
			},
		},
	)
}
