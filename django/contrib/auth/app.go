package auth

import (
	"database/sql"
	"net/http"

	"github.com/Nigel2392/django"
	"github.com/Nigel2392/django/apps"
	models "github.com/Nigel2392/django/contrib/auth/auth-models"
	core "github.com/Nigel2392/django/core"
	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/core/command"
	"github.com/Nigel2392/django/core/except"
	"github.com/Nigel2392/django/core/urls"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/goldcrest"
	"github.com/Nigel2392/mux"
	"github.com/Nigel2392/mux/middleware/sessions"
	"github.com/alexedwards/scs/v2"

	_ "github.com/Nigel2392/django/contrib/auth/auth-models/auth-models-mysql"
	_ "github.com/Nigel2392/django/contrib/auth/auth-models/auth-models-sqlite"
)

type AuthApplication struct {
	*apps.AppConfig
	Queries        models.Querier
	Session        *scs.SessionManager
	LoginWithEmail bool
}

var Auth *AuthApplication

func NewAppConfig() django.AppConfig {
	var app = &AuthApplication{
		AppConfig: apps.NewAppConfig("auth"),
	}
	app.Deps = []string{"session"}
	app.Path = "auth/"
	app.Middlewares = []core.Middleware{
		core.NewMiddleware(AddUserMiddleware()),
	}
	app.Cmd = []command.Command{
		command_create_user,
		command_set_password,
	}
	app.URLPatterns = []core.URL{
		urls.Pattern(
			urls.P("/login", mux.POST, mux.GET),
			mux.NewHandler(viewUserLogin),
			"login",
		),
		urls.Pattern(
			urls.P("/logout", mux.POST),
			mux.NewHandler(viewUserLogout),
			"logout",
		),
		urls.Pattern(
			urls.P("/register", mux.POST, mux.GET),
			mux.NewHandler(viewUserRegister),
			"register",
		),
	}
	app.Init = func(settings django.Settings) error {

		loginWithEmail, ok := settings.Get("AUTH_EMAIL_LOGIN")
		if ok {
			app.LoginWithEmail = loginWithEmail.(bool)
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

		goldcrest.Register(
			django.HOOK_SERVER_ERROR, 0,
			authRequiredHook,
		)

		attrs.RegisterFormFieldType(models.Password(""), func(opts ...func(fields.Field)) fields.Field {
			var newOpts = []func(fields.Field){
				fields.HelpText("Enter your password"),
				fields.Required(true),
			}
			newOpts = append(newOpts, opts...)
			return NewPasswordField(ChrFlagDigit|ChrFlagLower|ChrFlagUpper|ChrFlagSpecial, newOpts...)
		})

		return nil
	}

	Auth = app

	return app
}

func Login(r *http.Request, u *models.User) *models.User {
	var session = sessions.Retrieve(r)
	except.Assert(session != nil, 500, "session is nil")

	var err = session.RenewToken()
	except.Assert(err == nil, 500, "failed to renew session token")

	u.IsLoggedIn = true

	session.Set(SESSION_COOKIE_NAME, u.ID)

	return u
}

func Logout(r *http.Request) error {
	var session = sessions.Retrieve(r)
	except.Assert(session != nil, 500, "session is nil")

	var err = session.Destroy()
	except.Assert(err == nil, 500, "failed to destroy session")

	return nil
}
