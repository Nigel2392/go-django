package revisions

import (
	"fmt"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

type RevisionsAppConfig struct {
	*apps.DBRequiredAppConfig
}

var app *RevisionsAppConfig

func App() *RevisionsAppConfig {
	if app == nil {
		panic("revisions app not initialized")
	}
	return app
}

func NewAppConfig() *RevisionsAppConfig {
	app = &RevisionsAppConfig{
		DBRequiredAppConfig: apps.NewDBAppConfig(
			"revisions",
		),
	}

	app.ModelObjects = []attrs.Definer{
		&Revision{},
	}

	app.Init = func(settings django.Settings, db drivers.Database) error {

		if !django.AppInstalled("migrator") {
			var schemaEditor, err = migrator.GetSchemaEditor(db.Driver())
			if err != nil {
				return fmt.Errorf("failed to get schema editor: %w", err)
			}

			var table = migrator.NewModelTable(&Revision{})
			if err := schemaEditor.CreateTable(table, true); err != nil {
				return fmt.Errorf("failed to create pages table: %w", err)
			}

			for _, index := range table.Indexes() {
				if err := schemaEditor.AddIndex(table, index, true); err != nil {
					return fmt.Errorf("failed to create index %s: %w", index.Name(), err)
				}
			}
		}

		return nil
	}

	return app
}
