package models_mysql

import (
	"context"
	"database/sql"

	"github.com/Nigel2392/go-django/src/contrib/pages/models"
	dj_models "github.com/Nigel2392/go-django/src/models"
	"github.com/go-sql-driver/mysql"

	_ "embed"
)

//go:embed schema.mysql.sql
var mysql_schema string

func init() {
	models.Register(
		mysql.MySQLDriver{}, &dj_models.BaseBackend[models.Querier]{
			CreateTableQuery: mysql_schema,
			NewQuerier: func(d *sql.DB) (models.Querier, error) {
				return New(d), nil
			},
			PreparedQuerier: func(ctx context.Context, d *sql.DB) (models.Querier, error) {
				return Prepare(ctx, d)
			},
		},
	)
}
