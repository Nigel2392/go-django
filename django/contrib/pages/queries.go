package pages

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Nigel2392/django/contrib/pages/models"

	_ "embed"
)

var _ models.DBQuerier = (*Querier)(nil)

// Will get set by django_app.go.(NewAppConfig)
var QuerySet func() models.DBQuerier

type Querier struct {
	models.Querier
	Db *sql.DB
}

func (q *Querier) DB() *sql.DB {
	return q.Db
}

func (q *Querier) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return q.Db.BeginTx(ctx, opts)
}

func CreateTable(db *sql.DB) error {
	var driver = db.Driver()
	var backend, ok = models.GetBackend(driver)
	if !ok {
		panic(fmt.Sprintf("no backend configured for %T", driver))
	}

	return backend.CreateTable(db)
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
		Db:      db,
	}, nil
}
