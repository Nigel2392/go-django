package main

import (
	"context"
	"errors"
	"os"

	"github.com/Nigel2392/go-django/examples/todoapp/todos"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/contrib/auth"
	"github.com/Nigel2392/go-django/src/contrib/messages"
	"github.com/Nigel2392/go-django/src/contrib/session"
	"github.com/Nigel2392/go-django/src/core/command"
	_ "github.com/Nigel2392/go-django/src/core/filesystem/mediafiles/fs"
	"github.com/Nigel2392/go-django/src/core/logger"

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
			auth.APPVAR_AUTH_EMAIL_LOGIN:  true,
			migrator.APPVAR_MIGRATION_DIR: "./.private/migrations-todoapp",
		}),
		django.AppLogger(&logger.Logger{
			Level:       logger.INF,
			OutputTime:  true,
			WrapPrefix:  logger.ColoredLogWrapper,
			OutputDebug: os.Stdout,
			OutputInfo:  os.Stdout,
			OutputWarn:  os.Stdout,
			OutputError: os.Stdout,
		}),
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
		if errors.Is(err, command.ErrShouldExit) {
			// This is expected when running commands like `makemigrations` or `migrate`
			return
		}

		panic(err)
	}

	if err := app.Serve(); err != nil {
		panic(err)
	}
}
