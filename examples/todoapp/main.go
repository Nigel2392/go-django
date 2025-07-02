package main

import (
	"context"

	"github.com/Nigel2392/go-django/examples/todoapp/todos"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/contrib/auth"
	"github.com/Nigel2392/go-django/src/contrib/messages"
	"github.com/Nigel2392/go-django/src/contrib/session"
	_ "github.com/Nigel2392/go-django/src/core/filesystem/mediafiles/fs"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	var db, err = drivers.Open(context.Background(), "sqlite3", "./.private/db.todoapp.sqlite3")
	if err != nil {
		panic(err)
	}

	var app = django.App(
		django.Configure(map[string]interface{}{
			django.APPVAR_ALLOWED_HOSTS:   []string{"*"},
			django.APPVAR_DEBUG:           false,
			django.APPVAR_HOST:            "127.0.0.1",
			django.APPVAR_PORT:            "8080",
			django.APPVAR_DATABASE:        db,
			django.APPVAR_RECOVERER:       false,
			auth.APP_AUTH_EMAIL_LOGIN:     true,
			migrator.APPVAR_MIGRATION_DIR: "./migrations-todoapp",
		}),
		// django.AppMiddleware(
		// middleware.DefaultLogger.Intercept,
		// ),
		django.Apps(
			session.NewAppConfig,
			auth.NewAppConfig,
			admin.NewAppConfig,
			messages.NewAppConfig,
			todos.NewAppConfig,
			migrator.NewAppConfig,
		),
	)

	err = app.Initialize()
	if err != nil {
		panic(err)
	}

	if err := app.Serve(); err != nil {
		panic(err)
	}
}
