package models_mysql

import (
	"context"
	"database/sql"

	"github.com/Nigel2392/django/contrib/pages/models"
	"github.com/go-sql-driver/mysql"

	_ "embed"
)

//go:embed schema.mysql.sql
var mysql_schema string

func init() {
	models.Register(
		mysql.MySQLDriver{}, &models.BaseBackend{
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
