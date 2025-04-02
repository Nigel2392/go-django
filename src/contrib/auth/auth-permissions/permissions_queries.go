package auth_permissions

import (
	"context"
	"database/sql"

	_ "github.com/Nigel2392/go-django/src/contrib/auth/auth-permissions/auth-permissions-mysql"
	_ "github.com/Nigel2392/go-django/src/contrib/auth/auth-permissions/auth-permissions-sqlite"
	permissions_models "github.com/Nigel2392/go-django/src/contrib/auth/auth-permissions/permissions-models"
	models "github.com/Nigel2392/go-django/src/models"
)

type DBQuerier interface {
	permissions_models.Querier
	DB() *sql.DB
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

type dbQuerier struct {
	db      *sql.DB
	backend models.Backend[permissions_models.Querier]
	permissions_models.Querier
}

func (q *dbQuerier) CreateTable() error {
	return q.backend.CreateTable(q.db)
}

func (q *dbQuerier) DB() *sql.DB {
	return q.db
}

func (q *dbQuerier) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return q.db.BeginTx(ctx, opts)
}

var queries *dbQuerier

func NewQueries(db *sql.DB) (*dbQuerier, error) {
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
		backend: backend,
		Querier: qs,
	}

	return queries, nil
}
