package auth_permissions

import (
	"context"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	_ "github.com/Nigel2392/go-django/src/contrib/auth/auth-permissions/auth-permissions-mysql"
	_ "github.com/Nigel2392/go-django/src/contrib/auth/auth-permissions/auth-permissions-sqlite"
	permissions_models "github.com/Nigel2392/go-django/src/contrib/auth/auth-permissions/permissions-models"
	models "github.com/Nigel2392/go-django/src/models"
)

type DBQuerier interface {
	permissions_models.Querier
	DB() drivers.Database
	Begin(ctx context.Context) (drivers.Transaction, error)
}

type dbQuerier struct {
	db      drivers.Database
	backend models.Backend[permissions_models.Querier]
	permissions_models.Querier
}

func (q *dbQuerier) CreateTable() error {
	return q.backend.CreateTable(q.db)
}

func (q *dbQuerier) DB() drivers.Database {
	return q.db
}

func (q *dbQuerier) Begin(ctx context.Context) (drivers.Transaction, error) {
	return q.db.Begin(ctx)
}

var queries *dbQuerier

func NewQueries(db drivers.Database) (*dbQuerier, error) {
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
