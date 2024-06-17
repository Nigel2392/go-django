package pages

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Nigel2392/django/contrib/pages/models"
	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/mattn/go-sqlite3"

	models_mysql "github.com/Nigel2392/django/contrib/pages/models-mysql"
	models_postgres "github.com/Nigel2392/django/contrib/pages/models-postgres"
	models_sqlite "github.com/Nigel2392/django/contrib/pages/models-sqlite"

	_ "embed"
)

var _ models.DBQuerier = (*Querier)(nil)

type Querier struct {
	models.Querier
	db *sql.DB
}

func (q *Querier) DB() *sql.DB {
	return q.db
}

func (q *Querier) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return q.db.BeginTx(ctx, opts)
}

var querySet models.DBQuerier

//go:embed sqlc/schema.mysql.sql
var mySQLCreateTable string

//go:embed sqlc/schema.sqlite3.sql
var sqliteCreateTable string

//go:embed sqlc/schema.postgres.sql
var postgresCreateTable string

func CreateTable(db *sql.DB) error {
	var ctx = context.Background()
	switch db.Driver().(type) {
	case *mysql.MySQLDriver:
		_, err := db.ExecContext(ctx, mySQLCreateTable)
		if err != nil {
			return err
		}
	case *sqlite3.SQLiteDriver:
		_, err := db.ExecContext(ctx, sqliteCreateTable)
		if err != nil {
			return err
		}
	case *stdlib.Driver:
		_, err := db.ExecContext(ctx, postgresCreateTable)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported driver: %T", db.Driver())
	}

	return nil
}

func QuerySet(db *sql.DB) models.DBQuerier {
	if db == nil && querySet != nil {
		return querySet
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

	if querySet == nil {
		querySet = &Querier{
			Querier: q,
			db:      db,
		}
	}

	return querySet
}

func PrepareQuerySet(ctx context.Context, db *sql.DB) (models.DBQuerier, error) {
	var (
		q   models.Querier
		err error
	)

	switch db.Driver().(type) {
	case *mysql.MySQLDriver:
		q, err = models_mysql.Prepare(ctx, db)
	case *sqlite3.SQLiteDriver:
		q, err = models_sqlite.Prepare(ctx, db)
	case *stdlib.Driver:
		q, err = models_postgres.Prepare(ctx, db)
	default:
		panic(fmt.Sprintf("unsupported driver: %T", db.Driver()))
	}

	if err != nil {
		return nil, err
	}

	return &Querier{
		Querier: q,
		db:      db,
	}, nil
}
