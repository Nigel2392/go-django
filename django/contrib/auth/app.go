package auth

import (
	"net/http"

	"github.com/Nigel2392/django"
	"github.com/Nigel2392/django/apps"
	models "github.com/Nigel2392/django/contrib/auth/auth-models"
	core "github.com/Nigel2392/django/core"
	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/django/core/except"
	"github.com/Nigel2392/mux/middleware/sessions"
	"github.com/alexedwards/scs/v2"
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

	app.Middlewares = []core.Middleware{
		core.NewMiddleware(AddUserMiddleware()),
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

		db, ok := dbInt.(models.DBTX)
		assert.True(ok, "DATABASE setting must adhere to auth-models.DBTX interface")

		Auth.Queries = models.New(db)
		Auth.Session = sess

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
	return session.Destroy()
}
