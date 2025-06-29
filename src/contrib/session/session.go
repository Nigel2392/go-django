package session

import (
	"embed"
	"fmt"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/mux/middleware/sessions"
	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/v2/memstore"
)

//go:embed migrations/*
var migrationFS embed.FS

func NewAppConfig() django.AppConfig {
	attrs.RegisterModel(&Session{})

	var app = apps.NewAppConfig("session")

	var sessionManager = scs.New()

	app.ModelObjects = []attrs.Definer{
		&Session{},
	}

	app.Init = func(settings django.Settings) error {

		settings.Set(
			django.APPVAR_SESSION_MANAGER, sessionManager,
		)

		var dbInt, ok = settings.Get(django.APPVAR_DATABASE)
		var db drivers.Database
		if !ok {
			goto memstore
		}

		db, ok = dbInt.(drivers.Database)
		assert.True(ok, "DATABASE setting must be of type drivers.Database")

		switch db.Driver().(type) {
		case *drivers.DriverMySQL, *drivers.DriverMariaDB, *drivers.DriverPostgres, *drivers.DriverSQLite:
			if !django.AppInstalled("migrator") {
				var schemaEditor, err = migrator.GetSchemaEditor(db.Driver())
				if err != nil {
					return fmt.Errorf("failed to get schema editor: %w", err)
				}

				var table = migrator.NewModelTable(&Session{})
				if err := schemaEditor.CreateTable(table, true); err != nil {
					return fmt.Errorf("failed to create sessions table: %w", err)
				}

				for _, index := range table.Indexes() {
					if err := schemaEditor.AddIndex(table, index, true); err != nil {
						return fmt.Errorf("failed to create index %s: %w", index.Name(), err)
					}
				}
			}

			logger.Debug("Using QueryStore for session storage")
			sessionManager.Store = NewQueryStore(db)
			return nil
		}

	memstore:
		logger.Debug("Using memstore for session storage")
		sessionManager.Store = memstore.New()
		return nil
	}

	app.Routing = func(m django.Mux) {
		m.Use(
			sessions.SessionMiddleware(sessionManager),
		)
	}

	return &migrator.MigratorAppConfig{
		AppConfig: app,
		MigrationFS: filesystem.Sub(
			migrationFS, "migrations/session",
		),
	}
}
