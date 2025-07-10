package main

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"net/mail"
	"os"

	"github.com/Nigel2392/go-django/examples/blogapp/blog"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/contrib/auth"
	"github.com/Nigel2392/go-django/src/contrib/editor"
	_ "github.com/Nigel2392/go-django/src/contrib/editor/features"
	"github.com/Nigel2392/go-django/src/contrib/editor/features/images"
	"github.com/Nigel2392/go-django/src/contrib/messages"
	"github.com/Nigel2392/go-django/src/contrib/pages"
	"github.com/Nigel2392/go-django/src/contrib/reports"
	auditlogs "github.com/Nigel2392/go-django/src/contrib/reports/audit_logs"
	"github.com/Nigel2392/go-django/src/models"

	// auditlogs_mysql "github.com/Nigel2392/go-django/src/contrib/reports/audit_logs/audit_logs_mysql"
	"github.com/Nigel2392/go-django/src/contrib/revisions"

	"github.com/Nigel2392/go-django/src/contrib/session"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/checks"
	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles"
	_ "github.com/Nigel2392/go-django/src/core/filesystem/mediafiles/fs"
	"github.com/Nigel2392/go-django/src/core/filesystem/staticfiles"
	"github.com/Nigel2392/go-django/src/core/logger"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	var db, err = drivers.Open(context.Background(), "sqlite3", "./.private/db.blogapp.sqlite3")
	if err != nil {
		panic(err)
	}

	var app = django.App(
		django.Configure(map[string]interface{}{
			django.APPVAR_ALLOWED_HOSTS:   []string{"*"},
			django.APPVAR_DEBUG:           true,
			django.APPVAR_HOST:            "127.0.0.1",
			django.APPVAR_PORT:            "8080",
			django.APPVAR_DATABASE:        db,
			auth.APPVAR_AUTH_EMAIL_LOGIN:  true,
			migrator.APPVAR_MIGRATION_DIR: "./.private/migrations-blogapp",

			// django.APPVAR_RECOVERER:       false,
		}),
		django.AppLogger(&logger.Logger{
			Level:       logger.DBG,
			OutputTime:  true,
			WrapPrefix:  logger.ColoredLogWrapper,
			OutputDebug: os.Stdout,
			OutputInfo:  os.Stdout,
			OutputWarn:  os.Stdout,
			OutputError: os.Stdout,
		}),
		// django.AppMiddleware(
		// middleware.DefaultLogger.Intercept,
		// ),
		django.Apps(
			session.NewAppConfig,
			auth.NewAppConfig,
			admin.NewAppConfig,
			messages.NewAppConfig,
			pages.NewAppConfig,
			revisions.NewAppConfig,
			auditlogs.NewAppConfig,
			reports.NewAppConfig,
			editor.NewAppConfig,
			blog.NewAppConfig,
			images.NewAppConfig(&images.Options{
				MediaBackend: mediafiles.GetDefault(),
				MediaDir:     "images-blogapp",
			}),
			migrator.NewAppConfig,
		),
	)

	checks.Shutup("model.cant_check", true)
	checks.Shutup("admin.model_not_fully_implemented", true)
	checks.Shutup("field.invalid_db_type", func(m checks.Message) bool {
		return m.Object.(attrs.Field).Name() == "GroupPermissions"
	})

	mediafiles.SetDefault("filesystem")

	pages.SetRoutePrefix("/pages")
	app.Mux.Any("/pages/*", http.StripPrefix("/pages", pages.Serve(
		http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete,
	)), "pages")

	// logger.SetLevel(
	// logger.ERR,
	// )

	err = app.Initialize()
	if err != nil {
		panic(err)
	}

	var user = &auth.User{}
	var e, _ = mail.ParseAddress("admin@localhost")
	user.Email = (*drivers.Email)(e)
	user.Username = "admin"
	user.IsAdministrator = true
	user.SetPassword("Administrator123")

	if _, err := models.SaveModel(context.Background(), user); err != nil {
		panic(fmt.Errorf("failed to create admin user: %w", err))
	}

	if len(os.Args) == 1 {
		blogPages, err := queries.GetQuerySet(&blog.BlogPage{}).All()
		if err != nil {
			panic(fmt.Errorf("failed to get blog pages: %w", err))
		}
		fmt.Println("Blog pages:", len(blogPages))
		for page := range blogPages.Objects() {
			fmt.Printf(" - %s (ID: %d, %d)\n", page.Title, page.ID(), page.PageNode.PageID)
		}

		err = staticfiles.Collect(func(path string, f fs.File) error {
			var stat, err = f.Stat()
			if err != nil {
				return err
			}
			fmt.Println("Collected", path, stat.Size())
			return nil
		})
		if err != nil {
			panic(err)
		}
	}

	if err := app.Serve(); err != nil {
		panic(err)
	}
}
