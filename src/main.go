package main

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"net/http"
	"strconv"

	"github.com/Nigel2392/django"
	"github.com/Nigel2392/django/contrib/admin"
	"github.com/Nigel2392/django/contrib/auth"
	models "github.com/Nigel2392/django/contrib/auth/auth-models"
	"github.com/Nigel2392/django/contrib/blocks"
	"github.com/Nigel2392/django/contrib/pages"
	_ "github.com/Nigel2392/django/contrib/pages/backend-sqlite"
	auditlogs "github.com/Nigel2392/django/contrib/reports/audit_logs"
	auditlogs_sqlite "github.com/Nigel2392/django/contrib/reports/audit_logs/audit_logs_sqlite"
	"github.com/Nigel2392/django/contrib/session"
	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/core/errs"
	"github.com/Nigel2392/django/core/logger"
	"github.com/Nigel2392/django/core/staticfiles"
	"github.com/Nigel2392/django/forms"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/django/forms/modelforms"
	"github.com/Nigel2392/django/views/list"
	"github.com/Nigel2392/src/blog"
	"github.com/Nigel2392/src/core"

	_ "github.com/Nigel2392/django/contrib/pages/backend-sqlite"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
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
			core.NewAppConfig,
			blocks.NewAppConfig,
			blog.NewAppConfig,
		),
	)

	pages.SetPrefix("/pages")
	app.Mux.Any("/pages/*", http.StripPrefix("/pages", pages.Serve(
		http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete,
	)), "pages")

	var _ = admin.RegisterApp(
		"Auth",
		admin.AppOptions{
			RegisterToAdminMenu: true,
			AppLabel:            fields.S("Authentication and Authorization"),
			AppDescription:      fields.S("Manage users and groups, control access to your site with permissions."),
			MenuLabel:           fields.S("Auth"),
		},
		admin.ModelOptions{
			Model:               &models.User{},
			RegisterToAdminMenu: true,
			Labels: map[string]func() string{
				"ID":              fields.S("ID"),
				"Email":           fields.S("Email"),
				"Username":        fields.S("Username"),
				"FirstName":       fields.S("First name"),
				"LastName":        fields.S("Last name"),
				"Password":        fields.S("Password"),
				"IsAdministrator": fields.S("Is administrator"),
				"IsActive":        fields.S("Is active"),
				"CreatedAt":       fields.S("Created at"),
				"UpdatedAt":       fields.S("Updated at"),
			},
			GetForID: func(identifier any) (attrs.Definer, error) {
				var id, ok = identifier.(int)
				if !ok {
					var u, err = strconv.Atoi(fmt.Sprint(identifier))
					if err != nil {
						return nil, err
					}
					id = u
				}
				var user, err = auth.Auth.Queries.RetrieveByID(
					context.Background(),
					uint64(id),
				)
				if err != nil {
					return nil, err
				}
				return user, nil
			},
			GetList: func(amount, offset uint, fields []string) ([]attrs.Definer, error) {
				var users, err = auth.Auth.Queries.Retrieve(
					context.Background(), int32(amount), int32(offset),
				)
				if err != nil {
					return nil, err
				}
				var items = make([]attrs.Definer, 0)
				for _, u := range users {
					var cpy = u
					items = append(items, cpy)
				}
				return items, nil
			},
			AddView: admin.FormViewOptions{
				ViewOptions: admin.ViewOptions{
					Exclude: []string{"ID", "CreatedAt", "UpdatedAt"},
				},
				Panels: []admin.Panel{
					admin.TitlePanel(
						admin.FieldPanel("Email"),
					),
					admin.FieldPanel("Username"),
					admin.MultiPanel(
						admin.FieldPanel("FirstName"),
						admin.FieldPanel("LastName"),
					),
					admin.FieldPanel("Password"),
					admin.FieldPanel("IsAdministrator"),
					admin.FieldPanel("IsActive"),
				},
			},
			EditView: admin.FormViewOptions{
				ViewOptions: admin.ViewOptions{
					Exclude: []string{"ID"},
				},
				FormInit: initAuthEditForm,
				Panels: []admin.Panel{
					admin.TitlePanel(
						admin.FieldPanel("Email"),
					),
					admin.FieldPanel("Username"),
					admin.MultiPanel(
						admin.FieldPanel("FirstName"),
						admin.FieldPanel("LastName"),
					),
					admin.FieldPanel("Password"),
					admin.FieldPanel("PasswordConfirm"),
					admin.FieldPanel("IsAdministrator"),
					admin.FieldPanel("IsActive"),
				},
			},
			ListView: admin.ListViewOptions{
				ViewOptions: admin.ViewOptions{
					Fields: []string{
						"ID",
						"Email",
						"IsAdministrator",
						"IsActive",
						"CreatedAt",
						"UpdatedAt",
					},
				},
				Columns: map[string]list.ListColumn[attrs.Definer]{
					"Email": list.LinkColumn(
						fields.S("Email"),
						"Email", func(defs attrs.Definitions, row attrs.Definer) string {
							return django.Reverse("admin:apps:model:edit", "Auth", "User", defs.Get("ID"))
						},
					),
				},
				PerPage: 25,
			},
		},
	)

	var err = app.Initialize()
	if err != nil {
		panic(err)
	}

	app.Log.SetLevel(logger.DBG)

	err = staticfiles.Collect(func(pah string, f fs.File) error {
		var stat, err = f.Stat()
		if err != nil {
			return err
		}
		fmt.Println("Collected", pah, stat.Size())
		return nil
	})
	if err != nil {
		panic(err)
	}

	if err := app.Serve(); err != nil {
		panic(err)
	}
}

func initAuthEditForm(instance attrs.Definer, form modelforms.ModelForm[attrs.Definer]) {
	form.Ordering([]string{
		"Email",
		"Username",
		"FirstName",
		"LastName",
		"IsAdministrator",
		"IsActive",
		"Password",
		"PasswordConfirm",
	})
	form.AddField("PasswordConfirm", auth.NewPasswordField(
		auth.ChrFlagDEFAULT,
		fields.Label("Password Confirm"),
		fields.HelpText("Enter the password again to confirm"),
		fields.Required(false),
	))
	form.SetValidators(func(f forms.Form) []error {
		var (
			cleaned      = f.CleanedData()
			password1Int = cleaned["Password"]
			password2Int = cleaned["PasswordConfirm"]
		)
		if password2Int == nil || password2Int == "" {
			return nil
		}
		var (
			password1 = password1Int.(auth.PasswordString)
			password2 = password2Int.(auth.PasswordString)
		)
		if password1 != "" && password2 != "" && password1 != password2 {
			return []error{errs.Error("Passwords do not match")}
		} else if password1 != "" && password2 != "" && password1 == password2 {
			models.SetPassword(instance.(*models.User), string(password1))
			cleaned["Password"] = string(instance.(*models.User).Password)
		}
		return nil
	})
}
