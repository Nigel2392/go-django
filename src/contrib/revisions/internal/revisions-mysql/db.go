package revisions_mysql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Nigel2392/go-django/src/contrib/revisions/internal/revisions_db"
	"github.com/Nigel2392/go-django/src/models"
	"github.com/go-sql-driver/mysql"

	_ "embed"
)

//go:embed revisions.schema.sql
var mysql_schema string

func init() {
	revisions_db.Register(
		mysql.MySQLDriver{}, &models.BaseBackend[revisions_db.Querier]{
			CreateTableQuery: mysql_schema,
			NewQuerier: func(d *sql.DB) (revisions_db.Querier, error) {
				return New(d), nil
			},
			PreparedQuerier: func(ctx context.Context, d *sql.DB) (revisions_db.Querier, error) {
				return Prepare(ctx, d)
			},
		},
	)
}

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

func New(db DBTX) *Queries {
	return &Queries{db: db}
}

func Prepare(ctx context.Context, db DBTX) (*Queries, error) {
	q := Queries{db: db}
	var err error
	if q.deleteRevisionStmt, err = db.PrepareContext(ctx, deleteRevision); err != nil {
		return nil, fmt.Errorf("error preparing query DeleteRevision: %w", err)
	}
	if q.getRevisionByIDStmt, err = db.PrepareContext(ctx, getRevisionByID); err != nil {
		return nil, fmt.Errorf("error preparing query GetRevisionByID: %w", err)
	}
	if q.getRevisionsByObjectIDStmt, err = db.PrepareContext(ctx, getRevisionsByObjectID); err != nil {
		return nil, fmt.Errorf("error preparing query GetRevisionsByObjectID: %w", err)
	}
	if q.insertRevisionStmt, err = db.PrepareContext(ctx, insertRevision); err != nil {
		return nil, fmt.Errorf("error preparing query InsertRevision: %w", err)
	}
	if q.listRevisionsStmt, err = db.PrepareContext(ctx, listRevisions); err != nil {
		return nil, fmt.Errorf("error preparing query ListRevisions: %w", err)
	}
	if q.updateRevisionStmt, err = db.PrepareContext(ctx, updateRevision); err != nil {
		return nil, fmt.Errorf("error preparing query UpdateRevision: %w", err)
	}
	return &q, nil
}

func (q *Queries) Close() error {
	var err error
	if q.deleteRevisionStmt != nil {
		if cerr := q.deleteRevisionStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing deleteRevisionStmt: %w", cerr)
		}
	}
	if q.getRevisionByIDStmt != nil {
		if cerr := q.getRevisionByIDStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getRevisionByIDStmt: %w", cerr)
		}
	}
	if q.getRevisionsByObjectIDStmt != nil {
		if cerr := q.getRevisionsByObjectIDStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getRevisionsByObjectIDStmt: %w", cerr)
		}
	}
	if q.insertRevisionStmt != nil {
		if cerr := q.insertRevisionStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing insertRevisionStmt: %w", cerr)
		}
	}
	if q.listRevisionsStmt != nil {
		if cerr := q.listRevisionsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing listRevisionsStmt: %w", cerr)
		}
	}
	if q.updateRevisionStmt != nil {
		if cerr := q.updateRevisionStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing updateRevisionStmt: %w", cerr)
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
	db                         DBTX
	tx                         *sql.Tx
	deleteRevisionStmt         *sql.Stmt
	getRevisionByIDStmt        *sql.Stmt
	getRevisionsByObjectIDStmt *sql.Stmt
	insertRevisionStmt         *sql.Stmt
	listRevisionsStmt          *sql.Stmt
	updateRevisionStmt         *sql.Stmt
}

func (q *Queries) WithTx(tx *sql.Tx) revisions_db.Querier {
	return &Queries{
		db:                         tx,
		tx:                         tx,
		deleteRevisionStmt:         q.deleteRevisionStmt,
		getRevisionByIDStmt:        q.getRevisionByIDStmt,
		getRevisionsByObjectIDStmt: q.getRevisionsByObjectIDStmt,
		insertRevisionStmt:         q.insertRevisionStmt,
		listRevisionsStmt:          q.listRevisionsStmt,
		updateRevisionStmt:         q.updateRevisionStmt,
	}
}
