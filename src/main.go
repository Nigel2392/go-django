package main

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"strconv"

	"github.com/Nigel2392/django"
	"github.com/Nigel2392/django/contrib/admin"
	"github.com/Nigel2392/django/contrib/auth"
	auth_models "github.com/Nigel2392/django/contrib/auth/auth-models"
	models "github.com/Nigel2392/django/contrib/auth/auth-models"
	"github.com/Nigel2392/django/contrib/blocks"
	auditlogs "github.com/Nigel2392/django/contrib/reports/audit_logs"
	"github.com/Nigel2392/django/contrib/session"
	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/core/errs"
	"github.com/Nigel2392/django/core/logger"
	"github.com/Nigel2392/django/core/staticfiles"
	"github.com/Nigel2392/django/forms"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/django/forms/modelforms"
	"github.com/Nigel2392/django/views/list"
	"github.com/Nigel2392/mux/middleware"
	"github.com/Nigel2392/src/core"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	var app = django.App(
		django.Configure(map[string]interface{}{
			"ALLOWED_HOSTS": []string{"*"},
			"DEBUG":         true,
			"HOST":          "127.0.0.1",
			"PORT":          "8080",
			"DATABASE": func() *sql.DB {
				var db, err = sql.Open("sqlite3", "./.private/db.sqlite3")
				if err != nil {
					panic(err)
				}
				return db
			}(),
		}),
		django.AppMiddleware(
			middleware.DefaultLogger.Intercept,
		),
		django.Apps(
			session.NewAppConfig,
			auth.NewAppConfig,
			admin.NewAppConfig,
			auditlogs.NewAppConfig,
			core.NewAppConfig,
			blocks.NewAppConfig,
		),
	)

	var _ = admin.RegisterApp(
		"Auth",
		admin.AppOptions{
			RegisterToAdminMenu: true,
			MenuLabel:           fields.S("Auth"),
		},
		admin.ModelOptions{
			RegisterToAdminMenu: true,
			Labels: map[string]func() string{
				"Email":     fields.S("Object Email"),
				"FirstName": fields.S("Object First Name"),
				"LastName":  fields.S("Object Last Name"),
			},
			ListView: admin.ListViewOptions{
				ViewOptions: admin.ViewOptions{
					Fields: []string{
						"ID",
						"Email",
						"FirstName",
						"LastName",
						"IsAdministrator",
						"IsActive",
					},
					Labels: map[string]func() string{
						"Email":     fields.S("Object ListView Email"),
						"FirstName": fields.S("Object ListView First Name"),
						"LastName":  fields.S("Object ListView Last Name"),
					},
				},
				Columns: map[string]list.ListColumn[attrs.Definer]{
					"Email": list.LinkColumn[attrs.Definer](
						fields.S("Email"),
						"Email", func(defs attrs.Definitions, row attrs.Definer) string {
							return django.Reverse("admin:apps:model:edit", "Auth", "User", defs.Get("ID"))
						},
					),
				},
				PerPage: 25,
			},
			AddView: admin.FormViewOptions{
				ViewOptions: admin.ViewOptions{
					Labels: map[string]func() string{
						"Email":     fields.S("Object AddView Email"),
						"FirstName": fields.S("Object AddView First Name"),
						"LastName":  fields.S("Object AddView Last Name"),
					},
					Exclude: []string{"ID", "CreatedAt", "UpdatedAt"},
				},
			},
			EditView: admin.FormViewOptions{
				ViewOptions: admin.ViewOptions{
					Labels: map[string]func() string{
						"Email":     fields.S("Object EditView Email"),
						"FirstName": fields.S("Object EditView First Name"),
						"LastName":  fields.S("Object EditView Last Name"),
					},
				},
				FormInit: func(instance attrs.Definer, form modelforms.ModelForm[attrs.Definer]) {
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
						fields.Label("Password Confirm"),
						fields.HelpText("Enter the password again to confirm"),
						fields.Required(false),
						fields.MaxLength(64),
						auth.ValidateCharacters(false, auth.ChrFlagDigit|auth.ChrFlagLower|auth.ChrFlagUpper|auth.ChrFlagSpecial),
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
				},
			},
			Model: &auth_models.User{},
			GetForID: func(identifier any) (attrs.Definer, error) {
				var id, ok = identifier.(int)
				if !ok {
					var u, err = strconv.Atoi(fmt.Sprint(identifier))
					if err != nil {
						return nil, err
					}
					id = u
				}
				var user, err = auth.Auth.Queries.UserByID(
					context.Background(),
					uint64(id),
				)
				if err != nil {
					return nil, err
				}
				return &user, nil
			},
			GetList: func(amount, offset uint, fields []string) ([]attrs.Definer, error) {
				var users, err = auth.Auth.Queries.GetUsersWithPagination(
					context.Background(), uint64(amount), uint64(offset),
				)
				if err != nil {
					return nil, err
				}
				var items = make([]attrs.Definer, 0)
				for _, u := range users {
					var cpy = u
					items = append(items, &cpy)
				}
				return items, nil
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
