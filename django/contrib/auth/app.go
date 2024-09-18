package auth

import (
	"database/sql"
	"net/http"

	"github.com/Nigel2392/django"
	"github.com/Nigel2392/django/apps"
	"github.com/Nigel2392/django/contrib/admin"
	models "github.com/Nigel2392/django/contrib/auth/auth-models"
	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/core/command"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/mux"
	"github.com/alexedwards/scs/v2"

	_ "github.com/Nigel2392/django/contrib/auth/auth-models/auth-models-mysql"
	_ "github.com/Nigel2392/django/contrib/auth/auth-models/auth-models-sqlite"
)

type AuthApplication struct {
	*apps.AppConfig
	Queries        models.DBQuerier
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

		var q, err = models.NewQueries(db)
		assert.Err(err)

		Auth.Queries = q
		Auth.Session = sess

		admin.AdminSite.LogoutFunc = Logout
		admin.AdminSite.GetLoginForm = func(req *http.Request) admin.LoginForm {
			return UserLoginForm(req)
		}

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
