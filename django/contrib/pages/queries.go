package pages

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Nigel2392/django/contrib/pages/models"
	models_mysql "github.com/Nigel2392/django/contrib/pages/models-mysql"
	models_postgres "github.com/Nigel2392/django/contrib/pages/models-postgres"
	models_sqlite "github.com/Nigel2392/django/contrib/pages/models-sqlite"
	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/mattn/go-sqlite3"
)

var _ models.DBQuerier = (*Querier)(nil)

type Querier struct {
	models.Querier
	db *sql.DB
}

func (q *Querier) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return q.db.BeginTx(ctx, opts)
}

func QuerySet(db *sql.DB) models.DBQuerier {
	if db == nil && querier != nil {
		return querier
	}

	if db == nil {
		panic("db is nil")
	}

	var q models.Querier
	switch db.Driver().(type) {
	case *mysql.MySQLDriver:
		q = models_mysql.New(db)
	case *sqlite3.SQLiteDriver:
		q = models_sqlite.New(db)
	case *stdlib.Driver:
		q = models_postgres.New(db)
	default:
		panic(fmt.Sprintf("unsupported driver: %T", db.Driver()))
	}

	if querier == nil {
		querier = &Querier{
			Querier: q,
			db:      db,
		}
	}

	return querier
}
