package auth

import (
	"context"
	"embed"
	"fmt"
	"net/http"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	autherrors "github.com/Nigel2392/go-django/src/contrib/auth/auth_errors"
	"github.com/Nigel2392/go-django/src/contrib/auth/users"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/checks"
	"github.com/Nigel2392/go-django/src/core/command"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
	"github.com/Nigel2392/go-django/src/views/list"
	"github.com/Nigel2392/mux"
	"github.com/Nigel2392/mux/middleware/authentication"
	"github.com/alexedwards/scs/v2"
)

// The AuthApplication struct is the main struct used for the auth app.
type AuthApplication struct {
	*apps.AppConfig
	Session        *scs.SessionManager
	LoginWithEmail bool
}

var Auth *AuthApplication = &AuthApplication{}

//go:embed migrations/*
var migrationFS embed.FS

func NewAppConfig() django.AppConfig {
	var app = &AuthApplication{
		AppConfig: apps.NewAppConfig("auth"),
	}
	app.Deps = []string{"session"}
	app.Cmd = []command.Command{
		command_create_user,
		command_change_user,
		command_set_password,
	}
	app.ModelObjects = []attrs.Definer{
		&User{},
		&users.Group{},
		&users.Permission{},
		&users.UserGroup{},
		&users.GroupPermission{},
		&users.UserPermission{},
	}
	app.Routing = func(m django.Mux) {
		m.Use(
			AddUserMiddleware(),
		)

		if django.ConfigGet(django.Global.Settings, APPVAR_REGISTER_AUTH_URLS, true) {
			var g = m.Any("/auth", nil, "auth")
			g.Handle(mux.GET, "/login", mux.NewHandler(viewUserLogin), "login")
			g.Handle(mux.POST, "/login", mux.NewHandler(viewUserLogin))

			g.Handle(mux.GET, "/register", mux.NewHandler(viewUserRegister), "register")
			g.Handle(mux.POST, "/register", mux.NewHandler(viewUserRegister))

			g.Handle(mux.POST, "/logout", mux.NewHandler(LogoutView), "logout")
		}
	}
	app.Init = func(settings django.Settings) error {

		loginWithEmail, ok := settings.Get(APPVAR_AUTH_EMAIL_LOGIN)
		if ok {
			Auth.LoginWithEmail = loginWithEmail.(bool)
		}

		sessInt, ok := settings.Get(django.APPVAR_SESSION_MANAGER)
		assert.True(ok, "%s setting is required for 'auth' app", django.APPVAR_SESSION_MANAGER)

		sess, ok := sessInt.(*scs.SessionManager)
		assert.True(ok, "%s setting must be of type *scs.SessionManager", django.APPVAR_SESSION_MANAGER)

		dbInt, ok := settings.Get(django.APPVAR_DATABASE)
		assert.True(ok, "DATABASE setting is required for 'auth' app")

		db, ok := dbInt.(drivers.Database)
		assert.True(ok, "DATABASE setting must adhere to auth-drivers.DB interface")

		if !django.AppInstalled("migrator") {
			var schemaEditor, err = migrator.GetSchemaEditor(db.Driver())
			if err != nil {
				return fmt.Errorf("failed to get schema editor: %w", err)
			}

			var table = migrator.NewModelTable(&User{})
			if err := schemaEditor.CreateTable(context.Background(), table, true); err != nil {
				return fmt.Errorf("failed to create pages table: %w", err)
			}

			for _, index := range table.Indexes() {
				if err := schemaEditor.AddIndex(context.Background(), table, index, true); err != nil {
					return fmt.Errorf("failed to create index %s: %w", index.Name(), err)
				}
			}
		}

		Auth.Session = sess
		Auth.CtxProcessors = []func(ctx.ContextWithRequest){}

		// Set the user in the context if a request is present in the context.
		tpl.RequestProcessors(func(rc ctx.ContextWithRequest) {
			rc.Set("User",
				authentication.Retrieve(
					rc.Request(),
				),
			)
		})

		// Register hooks for authentication errors.
		//
		// These will intercept the server errors and allow for
		// custom handling of authentication errors.
		autherrors.RegisterHook("auth:login")

		// Configure the admin app for logins and logouts with the appropriate
		// user model.
		admin.ConfigureAuth(admin.AuthConfig{
			GetLoginForm: func(r *http.Request, formOpts ...func(forms.Form)) admin.LoginForm {
				return UserLoginForm(r, formOpts...)
			},
			Logout: Logout,
		})

		// Register the user model with the contenttypes package.
		//
		// This allows for the user model to be used in the admin app,
		// as well as in other apps that require it.
		contenttypes.Register(&contenttypes.ContentTypeDefinition{
			ContentObject:  &User{},
			GetLabel:       trans.S("User"),
			GetPluralLabel: trans.S("Users"),
			GetDescription: trans.S("User model for authentication"),
		})

		// Register the user model and the auth app with the admin app.
		var _ = admin.RegisterApp(
			"Auth",
			// Register the auth app with the admin app.
			admin.AppOptions{
				RegisterToAdminMenu: true,
				AppLabel:            trans.S("Authentication and Authorization"),
				AppDescription:      trans.S("Manage users and groups, control access to your site with permissions."),
				MenuLabel:           trans.S("Auth"),
				MenuOrder:           995,
				MenuIcon: func() string {
					return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-shield-exclamation" viewBox="0 0 16 16">
  <path d="M5.338 1.59a61 61 0 0 0-2.837.856.48.48 0 0 0-.328.39c-.554 4.157.726 7.19 2.253 9.188a10.7 10.7 0 0 0 2.287 2.233c.346.244.652.42.893.533q.18.085.293.118a1 1 0 0 0 .101.025 1 1 0 0 0 .1-.025q.114-.034.294-.118c.24-.113.547-.29.893-.533a10.7 10.7 0 0 0 2.287-2.233c1.527-1.997 2.807-5.031 2.253-9.188a.48.48 0 0 0-.328-.39c-.651-.213-1.75-.56-2.837-.855C9.552 1.29 8.531 1.067 8 1.067c-.53 0-1.552.223-2.662.524zM5.072.56C6.157.265 7.31 0 8 0s1.843.265 2.928.56c1.11.3 2.229.655 2.887.87a1.54 1.54 0 0 1 1.044 1.262c.596 4.477-.787 7.795-2.465 9.99a11.8 11.8 0 0 1-2.517 2.453 7 7 0 0 1-1.048.625c-.28.132-.581.24-.829.24s-.548-.108-.829-.24a7 7 0 0 1-1.048-.625 11.8 11.8 0 0 1-2.517-2.453C1.928 10.487.545 7.169 1.141 2.692A1.54 1.54 0 0 1 2.185 1.43 63 63 0 0 1 5.072.56"/>
  <path d="M7.001 11a1 1 0 1 1 2 0 1 1 0 0 1-2 0M7.1 4.995a.905.905 0 1 1 1.8 0l-.35 3.507a.553.553 0 0 1-1.1 0z"/>
</svg>`
				},
				// EnableIndexView: true,
			},
			// Register the user model with the admin app.
			admin.ModelOptions{
				Model:               &User{},
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
				// Customize the view / fields for the user models' create view.
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
				// Customize the view / fields for the user models' edit view.
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
						admin.FieldPanel("Permissions"),
						admin.FieldPanel("Groups"),
					},
				},
				// Customize the view / fields for the user models' list view.
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
							"Email", func(r *http.Request, defs attrs.Definitions, row attrs.Definer) string {
								return django.Reverse("admin:apps:model:edit", "Auth", "User", defs.Get("ID"))
							},
						),
					},
					PerPage: 25,
				},
			},
			admin.ModelOptions{
				Name:                "Groups",
				Model:               &users.Group{},
				RegisterToAdminMenu: true,
				MenuOrder:           2,
			},
			admin.ModelOptions{
				Name:                "Permissions",
				Model:               &users.Permission{},
				RegisterToAdminMenu: true,
				MenuOrder:           3,
			},
		)

		// Register the auth apps' password field with go-django.
		attrs.RegisterFormFieldType(NewPassword(""), func(opts ...func(fields.Field)) fields.Field {
			var newOpts = []func(fields.Field){
				fields.HelpText("Enter your password"),
				fields.Required(true),
			}
			newOpts = append(newOpts, opts...)
			return NewPasswordField(PasswordFieldOptions{
				Flags:         ChrFlagDEFAULT,
				IsRegistering: true,
			}, newOpts...)
		})

		return nil
	}

	*Auth = *app

	return &migrator.MigratorAppConfig{
		AppConfig: app,
		MigrationFS: filesystem.Sub(
			migrationFS, "migrations/auth",
		),
	}
}

func (app *AuthApplication) Check(ctx context.Context, settings django.Settings) []checks.Message {
	var messages = app.AppConfig.Check(ctx, settings)

	//	APPVAR_AUTH_EMAIL_LOGIN
	//	APPVAR_REGISTER_AUTH_URLS
	//	APPVAR_LOGIN_REDIRECT_URL

	if _, ok := settings.Get(APPVAR_LOGIN_REDIRECT_URL); !ok {
		messages = append(messages, checks.Warningf(
			"auth.login_redirect_url_not_set",
			"Login redirect URL is not set, using default value %q",
			nil, fmt.Sprintf("Set the %q setting to change the login redirect URL",
				APPVAR_LOGIN_REDIRECT_URL,
			),
			DEFAULT_LOGIN_REDIRECT_URL,
		))
	}

	return messages
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
		"Permissions",
		"Groups",
	})

	form.AddField("PasswordConfirm", NewPasswordField(
		PasswordFieldOptions{
			Flags:             ChrFlagDEFAULT,
			IsRegistering:     true,
			UseDefaultOptions: false,
		},
		fields.Label("Password Confirm"),
		fields.HelpText("Enter the password again to confirm"),
		fields.Required(false),
	))

	var user = instance.(*User)
	form.SetValidators(func(f forms.Form, cleaned map[string]interface{}) []error {

		var (
			password1Int = cleaned["Password"]
			password2Int = cleaned["PasswordConfirm"]
			password1, _ = password1Int.(*Password)
			password2, _ = password2Int.(*Password)
		)
		if password2 == nil || password2.Raw == "" {
			if password1 != nil && password1.Raw == user.Password.Hash {
				delete(cleaned, "Password")
				delete(cleaned, "PasswordConfirm")
				return nil
			}

			return []error{errors.Wrap(
				autherrors.ErrPwdNoMatch,
				"Password confirmation is required when changing the password",
			)}
		}

		if password1.Raw != "" && password2.Raw != "" && password1.Raw != password2.Raw {
			return []error{autherrors.ErrPwdNoMatch}
		} else if password1.Raw != "" && password2.Raw != "" && password1.Raw == password2.Raw {
			cleaned["Password"] = NewPassword(password1.Raw)
		}
		return nil
	})
}
