package main

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"net/http"

	"github.com/Nigel2392/django"
	"github.com/Nigel2392/django/contrib/admin"
	"github.com/Nigel2392/django/contrib/auth"
	"github.com/Nigel2392/django/contrib/blocks"
	"github.com/Nigel2392/django/contrib/pages"
	"github.com/Nigel2392/django/contrib/reports"
	auditlogs "github.com/Nigel2392/django/contrib/reports/audit_logs"
	auditlogs_sqlite "github.com/Nigel2392/django/contrib/reports/audit_logs/audit_logs_sqlite"
	"github.com/Nigel2392/django/contrib/session"
	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/core/filesystem/staticfiles"
	"github.com/Nigel2392/django/core/logger"
	"github.com/Nigel2392/django/forms/widgets"
	"github.com/Nigel2392/src/blog"
	"github.com/Nigel2392/src/core"
	"github.com/Nigel2392/src/todos"

	_ "github.com/Nigel2392/django/contrib/pages/backend-sqlite"
	_ "github.com/mattn/go-sqlite3"
)

type myInt int

type MyModel struct {
	ID   int
	Name string
	Bio  string
	Age  myInt
}

func (m *MyModel) FieldDefs() attrs.Definitions {
	return attrs.Define(m,
		attrs.NewField(m, "ID", &attrs.FieldConfig{
			Primary:  true,
			ReadOnly: true,
			Label:    "ID",
			HelpText: "The unique identifier of the model",
		}),
		attrs.NewField(m, "Name", &attrs.FieldConfig{
			Label:    "Name",
			HelpText: "The name of the model",
		}),
		attrs.NewField(m, "Bio", &attrs.FieldConfig{
			Label:    "Biography",
			HelpText: "The biography of the model",
			FormWidget: func(cfg attrs.FieldConfig) widgets.Widget {
				return widgets.NewTextarea(nil)
			},
		}),
		attrs.NewField(m, "Age", &attrs.FieldConfig{
			Label:    "Age",
			HelpText: "The age of the model",
			Validators: []func(interface{}) error{
				func(v interface{}) error {
					if v.(myInt) <= myInt(0) {
						return errors.New("Age must be greater than 0") //lint:ignore ST1005 Example.
					}
					return nil
				},
			},
		}),
	)
}

type MyTopLevelModel struct {
	MyModel
	Address string
}

func (m *MyTopLevelModel) FieldDefs() attrs.Definitions {
	var fields = m.MyModel.FieldDefs().Fields()
	fields = append(fields, attrs.NewField(m, "Address", &attrs.FieldConfig{
		Label:    "Address",
		HelpText: "The address of the model",
	}))
	return attrs.Define(m, fields...)
}

func main() {

	var m = &MyModel{}

	fmt.Println(attrs.Set(m, "Age", 0))
	fmt.Println(attrs.Set(m, "Age", -1))
	fmt.Println(attrs.Set(m, "Age", 1))

	var t = &MyTopLevelModel{}
	fmt.Println(attrs.Set(t, "Age", 0))
	fmt.Println(attrs.Set(t, "Age", -1))
	fmt.Println(attrs.Set(t, "Age", 1))

	var app = django.App(
		django.Configure(map[string]interface{}{
			"ALLOWED_HOSTS": []string{"*"},
			"DEBUG":         false,
			"HOST":          "127.0.0.1",
			"PORT":          "8080",
			"DATABASE": func() *sql.DB {
				// var db, err = sql.Open("mysql", "root:my-secret-pw@tcp(127.0.0.1:3306)/django-pages-test?parseTime=true&multiStatements=true")
				var db, err = sql.Open("sqlite3", "./.private/db.sqlite3")
				if err != nil {
					panic(err)
				}
				auditlogs.RegisterBackend(
					auditlogs_sqlite.NewSQLiteStorageBackend(db),
				)
				return db
			}(),

			"AUTH_EMAIL_LOGIN": true,
		}),
		// django.AppMiddleware(
		// middleware.DefaultLogger.Intercept,
		// ),
		django.Apps(
			session.NewAppConfig,
			auth.NewAppConfig,
			admin.NewAppConfig,
			pages.NewAppConfig,
			auditlogs.NewAppConfig,
			reports.NewAppConfig,
			core.NewAppConfig,
			blocks.NewAppConfig,
			blog.NewAppConfig,
			todos.NewAppConfig,
		),
	)

	pages.SetPrefix("/pages")
	app.Mux.Any("/pages/*", http.StripPrefix("/pages", pages.Serve(
		http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete,
	)), "pages")

	var err = app.Initialize()
	if err != nil {
		panic(err)
	}

	app.Log.SetLevel(logger.DBG)

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

	if err := app.Serve(); err != nil {
		panic(err)
	}
}
