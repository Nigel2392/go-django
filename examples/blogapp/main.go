package main

import (
	"context"
	"encoding/json"
	"fmt"
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
	"github.com/Nigel2392/go-django/src/contrib/documents"
	"github.com/Nigel2392/go-django/src/contrib/editor"
	_ "github.com/Nigel2392/go-django/src/contrib/editor/features"
	_ "github.com/Nigel2392/go-django/src/contrib/editor/features/images"
	_ "github.com/Nigel2392/go-django/src/contrib/editor/features/links"
	images "github.com/Nigel2392/go-django/src/contrib/images"
	"github.com/Nigel2392/go-django/src/contrib/messages"
	"github.com/Nigel2392/go-django/src/contrib/pages"
	"github.com/Nigel2392/go-django/src/contrib/reports"
	auditlogs "github.com/Nigel2392/go-django/src/contrib/reports/audit_logs"
	_ "github.com/Nigel2392/go-django/src/contrib/search/backends/searchqueries"
	"github.com/Nigel2392/go-django/src/contrib/settings"
	"github.com/Nigel2392/go-django/src/contrib/translations"
	"github.com/Nigel2392/go-django/src/core/secrets"
	_ "github.com/Nigel2392/go-django/src/core/secrets"
	"github.com/Nigel2392/mux"
	"github.com/google/uuid"

	"github.com/Nigel2392/go-django/src/contrib/revisions"

	"github.com/Nigel2392/go-django/src/contrib/session"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/checks"
	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles"
	mediafs "github.com/Nigel2392/go-django/src/core/filesystem/mediafiles/fs"

	"github.com/Nigel2392/go-django/src/core/logger"
)

func main() {
	os.MkdirAll("./.private/blogapp", 0755)

	var db, err = drivers.Open(context.Background(), "sqlite3", "./.private/blogapp/db.sqlite3")
	if err != nil {
		panic(err)
	}

	// This is the filesystem which will be used
	var mediaBackend = mediafs.NewBackend("./.private/blogapp/media", 0)
	mediafiles.RegisterBackend("filesystem", mediaBackend)
	mediafiles.SetDefault("filesystem")

	var app = django.App(
		django.Configure(map[string]interface{}{
			django.APPVAR_ALLOWED_HOSTS:   []string{"*"},
			django.APPVAR_DEBUG:           true,
			django.APPVAR_HOST:            "127.0.0.1",
			django.APPVAR_PORT:            "8080",
			django.APPVAR_DATABASE:        db,
			auth.APPVAR_AUTH_EMAIL_LOGIN:  true,
			migrator.APPVAR_MIGRATION_DIR: "./.private/blogapp/migrations",
			"APPVAR_SECRET_KEY":           "a very secret key",
			// translations.APPVAR_TRANSLATIONS_DEFAULT_LOCALE: language.Dutch,

			django.APPVAR_RECOVERER: false,
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
			settings.NewAppConfig,
			revisions.NewAppConfig,
			auditlogs.NewAppConfig,
			reports.NewAppConfig,
			editor.NewAppConfig,
			blog.NewAppConfig,
			images.NewAppConfig(&images.Options{
				MediaBackend: mediaBackend,
				MediaDir:     "images/blogpages",
			}),
			documents.NewAppConfig(&documents.Options{
				MediaBackend: mediaBackend,
				MediaDir:     "documents/blogpages",
			}),
			migrator.NewAppConfig,
			translations.NewAppConfig,
		),
	)

	secretKey := secrets.SECRET_KEY()
	signed, _ := secretKey.Sign(context.Background(), []byte("hello world"))
	fmt.Println("Secret key:", secretKey)
	fmt.Println(signed)
	unsigned, _ := secretKey.Unsign(context.Background(), signed)
	fmt.Println(string(unsigned))

	// Blog pages will be served from this route.
	pages.SetRoutePrefix("/pages")

	// Silence the following check messages
	checks.Shutup("model.cant_check", true)
	checks.Shutup("auth.login_redirect_url_not_set", true)
	checks.Shutup("admin.model_not_fully_implemented", true)
	checks.Shutup("field.invalid_db_type", func(m checks.Message) bool {
		return m.Object.(attrs.Field).Name() == "GroupPermissions"
	})

	// Register a route for chrome devtools
	// This is used to allow Chrome DevTools to connect to the app for debugging.
	// It serves a JSON file at /.well-known/appspecific/com.chrome.devtools.json
	// The file contains the workspace root and a UUID for the devtools session.
	// This allows for directly editing files in the app's directory from Chrome DevTools.
	//
	// Kinda cool? Yes. Useless? For sure.
	var devToolsID = uuid.NewString()
	app.Mux.Any("/.well-known/appspecific/com.chrome.devtools.json", mux.NewHandler(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		var wd, _ = os.Getwd()
		var devtools = map[string]map[string]string{
			"workspace": {
				"root": wd,
				"uuid": devToolsID,
			},
		}

		if err := json.NewEncoder(w).Encode(devtools); err != nil {
			logger.Errorf("Failed to encode devtools file: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}))

	// logger.SetLevel(
	// logger.ERR,
	// )

	err = app.Initialize()
	if err != nil {
		panic(err)
	}

	var created bool
	var user = &auth.User{}
	var e, _ = mail.ParseAddress("admin@localhost")
	user.Email = (*drivers.Email)(e)
	user.Username = "admin"
	user.IsAdministrator = true
	user.IsActive = true
	user.Password = auth.NewPassword("Administrator123!")

	if user, created, err = queries.GetQuerySet(&auth.User{}).Preload("EntrySet").Filter("Email", e.Address).GetOrCreate(user); err != nil {
		panic(fmt.Errorf("failed to create admin user: %w", err))
	}

	if created {
		logger.Infof("Admin user created: %v %s %s %t %t", user.ID, user.Username, user.Email, user.IsAdministrator, user.IsActive)
	} else {
		logger.Infof("Admin user already exists: %v %s %s %t %t", user.ID, user.Username, user.Email, user.IsAdministrator, user.IsActive)
	}

	var entrySet, ok = user.DataStore().GetValue("EntrySet")
	if !ok {
		panic("EntrySet not found in user data store")
	}
	entries, ok := entrySet.(*queries.RelRevFK[attrs.Definer])
	if !ok {
		panic(fmt.Errorf("EntrySet is not a slice of *auditlogs.Entry, got %T", entrySet))
	}

	for _, entry := range entries.AsList() {
		entry := entry.(*auditlogs.Entry)
		logger.Infof("Entry: %v %v %v %v", entry.ID, entry.Src, entry.Usr, entry.ObjectID)
	}

	if len(os.Args) == 1 {
		blogPages, err := queries.GetQuerySet(&blog.BlogPage{}).All()
		if err != nil {
			panic(fmt.Errorf("failed to get blog pages: %w", err))
		}
		fmt.Println("Blog pages:", len(blogPages))
		for page := range blogPages.Objects() {
			fmt.Printf(" - %q (ID: %d, %d)\n", page.Page.Title, page.ID(), page.Page.PageID)
		}

		pageRows, err := pages.NewPageQuerySet().
			Select("*").
			Specific().
			Search("test").
			Types(&blog.BlogPage{}).
			// Unpublished().
			All()
		if err != nil {
			panic(fmt.Errorf("failed to get pages: %w", err))
		}

		fmt.Println("Specific Pages:", len(pageRows))
		for _, pageRow := range pageRows {
			var specificPage = pageRow.Object
			var page = specificPage.Reference()
			fmt.Println(pageRow.Annotations)
			fmt.Printf(" - %q (ID: %d, %d)\n", page.Title, page.ID(), page.PageID)
			fmt.Printf("   - PageObject: %+v\n", specificPage.(*blog.BlogPage))
			// fmt.Printf("   - PageURL: %s\n", django.Reverse("pages", page.ID()))
		}

		//err = staticfiles.Collect(func(path string, f fs.File) error {
		//	var stat, err = f.Stat()
		//	if err != nil {
		//		return err
		//	}
		//	fmt.Println("Collected", path, stat.Size())
		//	return nil
		//})
		//if err != nil {
		//	panic(err)
		//}
	}

	if err := app.Serve(); err != nil {
		panic(err)
	}
}
