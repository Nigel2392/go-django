package auth

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	models "github.com/Nigel2392/go-django/src/contrib/auth/auth-models"
	auth_permissions "github.com/Nigel2392/go-django/src/contrib/auth/auth-permissions"
	permissions_models "github.com/Nigel2392/go-django/src/contrib/auth/auth-permissions/permissions-models"
	autherrors "github.com/Nigel2392/go-django/src/contrib/auth/auth_errors"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/command"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
	django_models "github.com/Nigel2392/go-django/src/models"
	"github.com/Nigel2392/go-django/src/permissions"
	"github.com/Nigel2392/go-django/src/views/list"
	"github.com/Nigel2392/mux"
	"github.com/Nigel2392/mux/middleware/authentication"
	"github.com/alexedwards/scs/v2"

	_ "github.com/Nigel2392/go-django/src/contrib/auth/auth-models/auth-models-mysql"
	_ "github.com/Nigel2392/go-django/src/contrib/auth/auth-models/auth-models-sqlite"
	_ "github.com/Nigel2392/go-django/src/contrib/auth/auth-permissions/auth-permissions-mysql"
	_ "github.com/Nigel2392/go-django/src/contrib/auth/auth-permissions/auth-permissions-sqlite"
)

type AuthApplication struct {
	*apps.AppConfig
	Queries        models.DBQuerier
	PermQueries    auth_permissions.DBQuerier
	Session        *scs.SessionManager
	LoginWithEmail bool
}

var Auth *AuthApplication = &AuthApplication{}

func NewAppConfig() django.AppConfig {
	var app = &AuthApplication{
		AppConfig: apps.NewAppConfig("auth"),
	}
	app.Deps = []string{"session"}
	app.Cmd = []command.Command{
		command_create_user,
		command_set_password,
	}
	app.Routing = func(m django.Mux) {
		var g = m.Any("/auth", nil, "auth")

		m.Use(
			AddUserMiddleware(),
		)

		g.Handle(mux.GET, "/login", mux.NewHandler(viewUserLogin))
		g.Handle(mux.POST, "/login", mux.NewHandler(viewUserLogin))

		g.Handle(mux.GET, "/register", mux.NewHandler(viewUserRegister))
		g.Handle(mux.POST, "/register", mux.NewHandler(viewUserRegister))

		g.Handle(mux.POST, "/logout", mux.NewHandler(viewUserLogout))

	}
	app.Init = func(settings django.Settings) error {

		loginWithEmail, ok := settings.Get("AUTH_EMAIL_LOGIN")
		if ok {
			Auth.LoginWithEmail = loginWithEmail.(bool)
		}

		sessInt, ok := settings.Get("SESSION_MANAGER")
		assert.True(ok, "SESSION_MANAGER setting is required for 'auth' app")

		sess, ok := sessInt.(*scs.SessionManager)
		assert.True(ok, "SESSION_MANAGER setting must adhere to scs.SessionManager interface")

		dbInt, ok := settings.Get("DATABASE")
		assert.True(ok, "DATABASE setting is required for 'auth' app")

		db, ok := dbInt.(*sql.DB)
		assert.True(ok, "DATABASE setting must adhere to auth-models.DBTX interface")

		var (
			q   models.DBQuerier
			pq  auth_permissions.DBQuerier
			b   django_models.Backend[models.Querier]
			pb  django_models.Backend[permissions_models.Querier]
			err error
		)

		if q, err = models.NewQueries(db); err != nil {
			return err
		}

		if pq, err = auth_permissions.NewQueries(db); err != nil {
			return err
		}

		if b, err = models.BackendForDB(db.Driver()); err != nil {
			return err
		}

		if pb, err = permissions_models.BackendForDB(db.Driver()); err != nil {
			return err
		}

		if err := b.CreateTable(db); err != nil {
			return err
		}

		if err := pb.CreateTable(db); err != nil {
			return err
		}

		Auth.Queries = q
		Auth.PermQueries = pq
		Auth.Session = sess
		Auth.CtxProcessors = []func(tpl.RequestContext){}

		permissions.Tester = auth_permissions.NewPermissionsBackend(
			pq,
		)

		tpl.Processors(func(rc tpl.RequestContext) {
			rc.Set("User",
				authentication.Retrieve(
					rc.Request(),
				),
			)
		})

		autherrors.RegisterHook()

		admin.ConfigureAuth(admin.AuthConfig{
			GetLoginForm: func(r *http.Request, formOpts ...func(forms.Form)) admin.LoginForm {
				return UserLoginForm(r, formOpts...)
			},
			Logout: Logout,
		})

		contenttypes.Register(&contenttypes.ContentTypeDefinition{
			ContentObject: &models.User{},
			GetInstance: func(i interface{}) (interface{}, error) {
				var id, ok = i.(int)
				if !ok {
					var u, err = strconv.Atoi(fmt.Sprint(i))
					if err != nil {
						return nil, err
					}
					id = u
				}
				var user, err = Auth.Queries.RetrieveByID(
					context.Background(),
					uint64(id),
				)
				if err != nil {
					return nil, err
				}
				return user, nil
			},
			GetInstances: func(amount, offset uint) ([]interface{}, error) {
				var users, err = Auth.Queries.Retrieve(
					context.Background(), int32(amount), int32(offset),
				)
				if err != nil {
					return nil, err
				}
				var items = make([]interface{}, 0)
				for _, u := range users {
					items = append(items, u)
				}
				return items, nil
			},
		})

		var _ = admin.RegisterApp(
			"Auth",
			admin.AppOptions{
				RegisterToAdminMenu: true,
				AppLabel:            trans.S("Authentication and Authorization"),
				AppDescription:      trans.S("Manage users and groups, control access to your site with permissions."),
				MenuLabel:           trans.S("Auth"),
			},
			admin.ModelOptions{
				Model:               &models.User{},
				RegisterToAdminMenu: true,
				Labels: map[string]func() string{
					"ID":              trans.S("ID"),
					"Email":           trans.S("Email"),
					"Username":        trans.S("Username"),
					"FirstName":       trans.S("First name"),
					"LastName":        trans.S("Last name"),
					"Password":        trans.S("Password"),
					"IsAdministrator": trans.S("Is administrator"),
					"IsActive":        trans.S("Is active"),
					"CreatedAt":       trans.S("Created at"),
					"UpdatedAt":       trans.S("Updated at"),
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
							trans.S("Email"),
							"Email", func(defs attrs.Definitions, row attrs.Definer) string {
								return django.Reverse("admin:apps:model:edit", "Auth", "User", defs.Get("ID"))
							},
						),
					},
					PerPage: 25,
				},
			},
		)

		attrs.RegisterFormFieldType(models.Password(""), func(opts ...func(fields.Field)) fields.Field {
			var newOpts = []func(fields.Field){
				fields.HelpText("Enter your password"),
				fields.Required(true),
			}
			newOpts = append(newOpts, opts...)
			return NewPasswordField(ChrFlagDEFAULT, true, newOpts...)
		})

		return nil
	}

	*Auth = *app

	return app
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
	form.AddField("PasswordConfirm", NewPasswordField(
		ChrFlagDEFAULT,
		true,
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
			password1 = password1Int.(PasswordString)
			password2 = password2Int.(PasswordString)
		)
		if password1 != "" && password2 != "" && password1 != password2 {
			return []error{autherrors.ErrPwdNoMatch}
		} else if password1 != "" && password2 != "" && password1 == password2 {
			SetPassword(instance.(*models.User), string(password1))
			cleaned["Password"] = string(instance.(*models.User).Password)
		}
		return nil
	})
}
