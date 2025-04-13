package openauth2_models_mysql

import (
	"context"
	"database/sql"

	openauth2models "github.com/Nigel2392/go-django/src/contrib/openauth2/openauth2_models"
	dj_models "github.com/Nigel2392/go-django/src/models"
	"github.com/go-sql-driver/mysql"

	_ "embed"
)

//go:embed sql/schema.sql
var mysql_schema string

func init() {
	openauth2models.Register(
		mysql.MySQLDriver{}, &dj_models.BaseBackend[openauth2models.Querier]{
			CreateTableQuery: mysql_schema,
			NewQuerier: func(d *sql.DB) (openauth2models.Querier, error) {
				return New(d), nil
			},
			PreparedQuerier: func(ctx context.Context, d *sql.DB) (openauth2models.Querier, error) {
				return Prepare(ctx, d)
			},
		},
	)
}
