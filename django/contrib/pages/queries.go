package pages

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Nigel2392/django/contrib/pages/models"

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

func CreateTable(db *sql.DB) error {
	var driver = db.Driver()
	var backend, ok = models.GetBackend(driver)
	if !ok {
		panic(fmt.Sprintf("no backend configured for %T", driver))
	}

	return backend.CreateTable(db)
}

func QuerySet(db *sql.DB) models.DBQuerier {
	if db == nil && querySet != nil {
		return querySet
	}

	if db == nil {
		panic("db is nil")
	}

	var driver = db.Driver()
	var backend, ok = models.GetBackend(driver)
	if !ok {
		panic(fmt.Sprintf("no backend configured for %T", driver))
	}

	var qs, err = backend.NewQuerySet(db)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize queryset for backend %T", backend))
	}

	if querySet == nil {
		querySet = &Querier{
			Querier: qs,
			db:      db,
		}
	}

	return querySet
}

func PrepareQuerySet(ctx context.Context, db *sql.DB) (models.DBQuerier, error) {
	var driver = db.Driver()
	var backend, ok = models.GetBackend(driver)
	if !ok {
		panic(fmt.Sprintf("no backend configured for %T", driver))
	}

	var qs, err = backend.Prepare(ctx, db)
	if err != nil {
		return nil, err
	}

	return &Querier{
		Querier: qs,
		db:      db,
	}, nil
}
