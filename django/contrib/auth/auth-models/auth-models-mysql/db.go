package models_mysql

import (
	"context"
	"database/sql"
	"fmt"

	_ "embed"

	models "github.com/Nigel2392/django/contrib/auth/auth-models"
	dj_models "github.com/Nigel2392/django/models"
	"github.com/go-sql-driver/mysql"
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
				return Prepare(ctx, nil)
			},
		},
	)
}

func New(db models.DBTX) *Queries {
	return &Queries{db: db}
}

func Prepare(ctx context.Context, db models.DBTX) (*Queries, error) {
	q := Queries{db: db}
	var err error
	if q.countStmt, err = db.PrepareContext(ctx, count); err != nil {
		return nil, fmt.Errorf("error preparing query Count: %w", err)
	}
	if q.countManyStmt, err = db.PrepareContext(ctx, countMany); err != nil {
		return nil, fmt.Errorf("error preparing query CountMany: %w", err)
	}
	if q.createUserStmt, err = db.PrepareContext(ctx, createUser); err != nil {
		return nil, fmt.Errorf("error preparing query CreateUser: %w", err)
	}
	if q.deleteUserStmt, err = db.PrepareContext(ctx, deleteUser); err != nil {
		return nil, fmt.Errorf("error preparing query DeleteUser: %w", err)
	}
	if q.retrieveStmt, err = db.PrepareContext(ctx, retrieve); err != nil {
		return nil, fmt.Errorf("error preparing query Retrieve: %w", err)
	}
	if q.retrieveByEmailStmt, err = db.PrepareContext(ctx, retrieveByEmail); err != nil {
		return nil, fmt.Errorf("error preparing query RetrieveByEmail: %w", err)
	}
	if q.retrieveByIDStmt, err = db.PrepareContext(ctx, retrieveByID); err != nil {
		return nil, fmt.Errorf("error preparing query RetrieveByID: %w", err)
	}
	if q.retrieveByUsernameStmt, err = db.PrepareContext(ctx, retrieveByUsername); err != nil {
		return nil, fmt.Errorf("error preparing query RetrieveByUsername: %w", err)
	}
	if q.retrieveManyStmt, err = db.PrepareContext(ctx, retrieveMany); err != nil {
		return nil, fmt.Errorf("error preparing query RetrieveMany: %w", err)
	}
	if q.updateUserStmt, err = db.PrepareContext(ctx, updateUser); err != nil {
		return nil, fmt.Errorf("error preparing query UpdateUser: %w", err)
	}
	return &q, nil
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
	switch {
	case stmt != nil && q.tx != nil:
		return q.tx.StmtContext(ctx, stmt).ExecContext(ctx, args...)
	case stmt != nil:
		return stmt.ExecContext(ctx, args...)
	default:
		return q.db.ExecContext(ctx, query, args...)
	}
}

func (q *Queries) query(ctx context.Context, stmt *sql.Stmt, query string, args ...interface{}) (*sql.Rows, error) {
	switch {
	case stmt != nil && q.tx != nil:
		return q.tx.StmtContext(ctx, stmt).QueryContext(ctx, args...)
	case stmt != nil:
		return stmt.QueryContext(ctx, args...)
	default:
		return q.db.QueryContext(ctx, query, args...)
	}
}

func (q *Queries) queryRow(ctx context.Context, stmt *sql.Stmt, query string, args ...interface{}) *sql.Row {
	switch {
	case stmt != nil && q.tx != nil:
		return q.tx.StmtContext(ctx, stmt).QueryRowContext(ctx, args...)
	case stmt != nil:
		return stmt.QueryRowContext(ctx, args...)
	default:
		return q.db.QueryRowContext(ctx, query, args...)
	}
}

type Queries struct {
	db                     models.DBTX
	tx                     *sql.Tx
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

func (q *Queries) WithTx(tx *sql.Tx) models.Querier {
	return &Queries{
		db:                     tx,
		tx:                     tx,
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
