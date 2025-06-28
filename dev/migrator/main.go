package main

import (
	"context"

	_ "github.com/mattn/go-sqlite3"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/openauth2"
	"github.com/Nigel2392/go-django/src/contrib/pages"
	"github.com/Nigel2392/go-django/src/contrib/session"
)

// utility application to easily generate migrations during development
//
// migrations should be placed in <app>/migrations/<app_name>/<model_name>/<migration_name>.mig
//
// the app where the migrations are placed should be wrapped by [migrator.MigratorAppConfig], or implement [migrator.MigrationAppConfig],
// which is used to provide the migration filesystem to the migrator engine

func main() {
	var app = django.App(
		django.Configure(map[string]interface{}{
			django.APPVAR_ALLOWED_HOSTS: []string{"*"},
			django.APPVAR_DEBUG:         false,
			django.APPVAR_HOST:          "127.0.0.1",
			django.APPVAR_PORT:          "8080",
			django.APPVAR_DATABASE: func() drivers.Database {
				// var db, err = drivers.Open("mysql", "root:my-secret-pw@tcp(127.0.0.1:3306)/django-pages-test?parseTime=true&multiStatements=true")
				var db, err = drivers.Open(context.Background(), "sqlite3", "./.private/db.sqlite3")
				if err != nil {
					panic(err)
				}
				return db
			}(),
		}),
		django.Apps(
			session.NewAppConfig,
			pages.NewAppConfig,
			openauth2.NewAppConfig(openauth2.Config{}),
			migrator.NewAppConfig,
		),
	)

	if err := app.Initialize(); err != nil {
		panic(err)
	}
}
