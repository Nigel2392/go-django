package revisions

import (
	"context"
	"embed"
	"fmt"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/filesystem"
)

//go:embed migrations/*
var migrationFS embed.FS

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

func NewAppConfig() django.AppConfig {
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
			if err := schemaEditor.CreateTable(context.Background(), table, true); err != nil {
				return fmt.Errorf("failed to create pages table: %w", err)
			}

			for _, index := range table.Indexes() {
				if err := schemaEditor.AddIndex(context.Background(), table, index, true); err != nil {
					return fmt.Errorf("failed to create index %s: %w", index.Name(), err)
				}
			}
		}

		admin.RegisterApp(
			"revisions",
			admin.AppOptions{
				RegisterToAdminMenu: false,
				EnableIndexView:     false,
			},
			admin.ModelOptions{
				Name:                "revision",
				Model:               &Revision{},
				RegisterToAdminMenu: false,
				DisallowList:        true,
				DisallowCreate:      true,
				DisallowEdit:        true,
				DisallowDelete:      true,
			},
		)

		return nil
	}

	return &migrator.MigratorAppConfig{
		AppConfig: app,
		MigrationFS: filesystem.Sub(
			migrationFS, "migrations/revisions",
		),
	}
}
