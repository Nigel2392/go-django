package main

import (
	"context"
	"fmt"
	"net/mail"
	"os"
	"runtime"
	"time"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/auth"
	"github.com/Nigel2392/go-django/contrib/blocks"
	"github.com/Nigel2392/go-django/contrib/editor"
	"github.com/Nigel2392/go-django/contrib/messages"
	"github.com/Nigel2392/go-django/contrib/reports"
	auditlogs "github.com/Nigel2392/go-django/contrib/reports/audit_logs"
	"github.com/Nigel2392/go-django/contrib/revisions"
	"github.com/Nigel2392/go-django/contrib/session"
	"github.com/Nigel2392/go-django/contrib/shop"
	"github.com/Nigel2392/go-django/examples/commerce/src/suppliers/dbsupplier"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles"
	mediafs "github.com/Nigel2392/go-django/src/core/filesystem/mediafiles/fs"
	"github.com/Nigel2392/go-django/src/core/logger"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	os.MkdirAll("./.private/commerce", 0755)
	//	os.Remove("./.private/commerce/db.sqlite3")
	//	dst, err := os.Create("./.private/commerce/db.sqlite3")
	//	if err != nil {
	//		panic(err)
	//	}
	//
	//	src, err := os.Open("./.private/commerce/db.sqlite3")
	//	if err != nil {
	//		panic(err)
	//	}
	//
	//	io.Copy(dst, src)
	//	dst.Close()
	//	src.Close()
	// This is the filesystem which will be used
	var mediaBackend = mediafs.NewBackend("./.private/commerce/media", 0)
	mediafiles.RegisterBackend("filesystem", mediaBackend)
	mediafiles.SetDefault("filesystem")

	db, err := drivers.Open(context.Background(), "sqlite3", "./.private/commerce/db.sqlite3")
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
			migrator.APPVAR_MIGRATION_DIR: "./.private/commerce/migrations",
		}),

		django.Apps(
			session.NewAppConfig,
			messages.NewAppConfig,
			auth.NewAppConfig,
			admin.NewAppConfig,
			auditlogs.NewAppConfig,
			reports.NewAppConfig,
			migrator.NewAppConfig,
			editor.NewAppConfig,
			blocks.NewAppConfig,
			shop.NewAppConfig,
			revisions.NewAppConfig,
			dbsupplier.NewAppConfig,
		),
	)
	go func() {
		for {
			time.Sleep(500 * time.Millisecond)
			runtime.GC()
		}
	}()
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
