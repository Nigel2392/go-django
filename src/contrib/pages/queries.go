package pages

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/Nigel2392/go-django/src/contrib/pages/backend-mysql"
	_ "github.com/Nigel2392/go-django/src/contrib/pages/backend-sqlite"
	"github.com/Nigel2392/go-django/src/contrib/pages/models"

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
	var backend, err = models.GetBackend(driver)
	if err != nil {
		panic(fmt.Errorf("no backend configured for %T: %w", driver, err))
	}

	return backend.CreateTable(db)
}

func PrepareQuerySet(ctx context.Context, db *sql.DB) (models.DBQuerier, error) {
	var driver = db.Driver()
	var backend, err = models.GetBackend(driver)
	if err != nil {
		panic(fmt.Errorf("no backend configured for %T: %w", driver, err))
	}

	qs, err := backend.Prepare(ctx, db)
	if err != nil {
		return nil, err
	}

	return &Querier{
		Querier: qs,
		Db:      db,
	}, nil
}
