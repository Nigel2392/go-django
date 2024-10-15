package auth_permissions

import (
	"context"
	"database/sql"

	permissions_models "github.com/Nigel2392/go-django/src/contrib/auth/auth-permissions/permissions-models"
)

type DBQuerier interface {
	permissions_models.Querier
	DB() *sql.DB
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

type dbQuerier struct {
	db *sql.DB
	permissions_models.Querier
}

func (q *dbQuerier) DB() *sql.DB {
	return q.db
}

func (q *dbQuerier) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return q.db.BeginTx(ctx, opts)
}

var queries DBQuerier

func NewQueries(db *sql.DB) (DBQuerier, error) {
	if queries != nil {
		return queries, nil
	}

	var backend, err = permissions_models.BackendForDB(db.Driver())
	if err != nil {
		return nil, err
	}

	qs, err := backend.NewQuerySet(db)
	if err != nil {
		return nil, err
	}

	queries = &dbQuerier{
		db:      db,
		Querier: qs,
	}

	return queries, nil
}
