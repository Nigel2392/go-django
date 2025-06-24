package models_mysql

import (
	"context"
	"database/sql"
	"fmt"

	_ "embed"

	"github.com/Nigel2392/go-django-queries/src/drivers"
	models "github.com/Nigel2392/go-django/src/contrib/auth/auth-models"
	dj_models "github.com/Nigel2392/go-django/src/models"
	"github.com/go-sql-driver/mysql"
)

//go:embed schema.mysql.sql
var mysql_schema string

func init() {
	models.Register(
		mysql.MySQLDriver{}, &dj_models.BaseBackend[models.Querier]{
			CreateTableQuery: mysql_schema,
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

func (q *Queries) Close() error {
	var err error
	if q.countStmt != nil {
		if cerr := q.countStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing countStmt: %w", cerr)
		}
	}
	if q.countManyStmt != nil {
		if cerr := q.countManyStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing countManyStmt: %w", cerr)
		}
	}
	if q.createUserStmt != nil {
		if cerr := q.createUserStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing createUserStmt: %w", cerr)
		}
	}
	if q.deleteUserStmt != nil {
		if cerr := q.deleteUserStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing deleteUserStmt: %w", cerr)
		}
	}
	if q.retrieveStmt != nil {
		if cerr := q.retrieveStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing retrieveStmt: %w", cerr)
		}
	}
	if q.retrieveByEmailStmt != nil {
		if cerr := q.retrieveByEmailStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing retrieveByEmailStmt: %w", cerr)
		}
	}
	if q.retrieveByIDStmt != nil {
		if cerr := q.retrieveByIDStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing retrieveByIDStmt: %w", cerr)
		}
	}
	if q.retrieveByUsernameStmt != nil {
		if cerr := q.retrieveByUsernameStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing retrieveByUsernameStmt: %w", cerr)
		}
	}
	if q.retrieveManyStmt != nil {
		if cerr := q.retrieveManyStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing retrieveManyStmt: %w", cerr)
		}
	}
	if q.updateUserStmt != nil {
		if cerr := q.updateUserStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing updateUserStmt: %w", cerr)
		}
	}
	return err
}

func (q *Queries) exec(ctx context.Context, stmt *sql.Stmt, query string, args ...interface{}) (sql.Result, error) {
	return q.db.ExecContext(ctx, query, args...)
}

func (q *Queries) query(ctx context.Context, stmt *sql.Stmt, query string, args ...interface{}) (drivers.SQLRows, error) {
	return q.db.QueryContext(ctx, query, args...)
}

func (q *Queries) queryRow(ctx context.Context, stmt *sql.Stmt, query string, args ...interface{}) drivers.SQLRow {
	return q.db.QueryRowContext(ctx, query, args...)
}

type Queries struct {
	db                     drivers.DB
	countStmt              *sql.Stmt
	countManyStmt          *sql.Stmt
	createUserStmt         *sql.Stmt
	deleteUserStmt         *sql.Stmt
	retrieveStmt           *sql.Stmt
	retrieveByEmailStmt    *sql.Stmt
	retrieveByIDStmt       *sql.Stmt
	retrieveByUsernameStmt *sql.Stmt
	retrieveManyStmt       *sql.Stmt
	updateUserStmt         *sql.Stmt
}

func (q *Queries) WithTx(tx drivers.Transaction) models.Querier {
	return &Queries{
		db:                     tx,
		countStmt:              q.countStmt,
		countManyStmt:          q.countManyStmt,
		createUserStmt:         q.createUserStmt,
		deleteUserStmt:         q.deleteUserStmt,
		retrieveStmt:           q.retrieveStmt,
		retrieveByEmailStmt:    q.retrieveByEmailStmt,
		retrieveByIDStmt:       q.retrieveByIDStmt,
		retrieveByUsernameStmt: q.retrieveByUsernameStmt,
		retrieveManyStmt:       q.retrieveManyStmt,
		updateUserStmt:         q.updateUserStmt,
	}
}
