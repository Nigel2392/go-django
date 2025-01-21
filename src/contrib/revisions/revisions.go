package revisions

import (
	"context"
	"database/sql"
	"fmt"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/contrib/revisions/internal/revisions_db"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/models"
)

type RevisionsAppConfig struct {
	*apps.DBRequiredAppConfig
	backend models.Backend[revisions_db.Querier]
}

var app *RevisionsAppConfig

func App() *RevisionsAppConfig {
	if app == nil {
		panic("revisions app not initialized")
	}
	return app
}

func NewRevisionsAppConfig() *RevisionsAppConfig {
	app = &RevisionsAppConfig{
		DBRequiredAppConfig: apps.NewDBAppConfig(
			"revisions",
		),
	}

	app.Init = func(settings django.Settings, db *sql.DB) error {

		var driver = db.Driver()
		var backend, err = revisions_db.GetBackend(driver)
		if err != nil {
			panic(fmt.Errorf(
				"no backend configured for %T: %w",
				driver, err,
			))
		}

		app.backend = backend

		return backend.CreateTable(db)
	}

	return app
}

func QuerySet(ctx context.Context) *RevisionQuerier {
	if ctx == nil {
		ctx = context.Background()
	}

	var app = App()
	var q, err = app.backend.NewQuerySet(app.DB)
	assert.Assert(
		err == nil,
		"error creating new query set: %v", err,
	)

	return &RevisionQuerier{
		ctx:     ctx,
		Querier: q,
	}
}
