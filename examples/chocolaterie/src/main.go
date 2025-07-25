package main

import (
	"context"
	"fmt"
	"net/mail"
	"os"

	chocolaterie "github.com/Nigel2392/go-django/examples/chocolaterie/src/chocolaterie"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/contrib/auth"
	"github.com/Nigel2392/go-django/src/contrib/editor"
	_ "github.com/Nigel2392/go-django/src/contrib/editor/features"
	"github.com/Nigel2392/go-django/src/contrib/messages"
	"github.com/Nigel2392/go-django/src/contrib/pages"
	"github.com/Nigel2392/go-django/src/contrib/reports"
	auditlogs "github.com/Nigel2392/go-django/src/contrib/reports/audit_logs"
	"github.com/Nigel2392/go-django/src/contrib/session"
	"github.com/Nigel2392/go-django/src/contrib/settings"
	"github.com/Nigel2392/go-django/src/core/logger"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	os.MkdirAll("./.private/chocolaterie", 0755)
	//
	var db, err = drivers.Open(context.Background(), "sqlite3", "./.private/chocolaterie/db.sqlite3")
	if err != nil {
		panic(err)
	}

	var app = django.App(
		django.Configure(map[string]interface{}{
			django.APPVAR_ALLOWED_HOSTS: []string{"*"},
			django.APPVAR_DEBUG:         false,
			django.APPVAR_HOST:          "127.0.0.1",
			django.APPVAR_PORT:          "8080",
			django.APPVAR_DATABASE:      db,
			django.APPVAR_RECOVERER:     false,

			auth.APPVAR_AUTH_EMAIL_LOGIN:  true,
			migrator.APPVAR_MIGRATION_DIR: "./.private/chocolaterie/migrations",
		}),

		django.Apps(
			session.NewAppConfig,
			messages.NewAppConfig,
			auth.NewAppConfig,
			admin.NewAppConfig,
			editor.NewAppConfig,
			pages.NewAppConfig,
			settings.NewAppConfig,
			auditlogs.NewAppConfig,
			reports.NewAppConfig,
			chocolaterie.NewAppConfig,
			migrator.NewAppConfig,
		),
	)

	pages.SetRoutePrefix("/pages")

	err = app.Initialize()
	if err != nil {
		panic(err)
	}

	app.Log.SetLevel(
		logger.DBG,
	)

	var created bool
	var user = &auth.User{}
	var e, _ = mail.ParseAddress("admin@localhost")
	user.Email = (*drivers.Email)(e)
	user.Username = "admin"
	user.IsAdministrator = true
	user.IsActive = true
	user.Password = auth.NewPassword("Administrator123!")

	if user, created, err = queries.GetQuerySet(&auth.User{}).Filter("Email", e.Address).GetOrCreate(user); err != nil {
		panic(fmt.Errorf("failed to create admin user: %w", err))
	}

	if created {
		logger.Infof("Admin user created: %v %s %s %t %t", user.ID, user.Username, user.Email, user.IsAdministrator, user.IsActive)
	} else {
		logger.Infof("Admin user already exists: %v %s %s %t %t", user.ID, user.Username, user.Email, user.IsAdministrator, user.IsActive)
	}

	if err := app.Serve(); err != nil {
		panic(err)
	}
}
