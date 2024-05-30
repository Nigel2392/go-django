package auth

import (
	"github.com/Nigel2392/django"
	"github.com/Nigel2392/django/apps"
	models "github.com/Nigel2392/django/contrib/auth/auth-models"
	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/django/core/http_"
	"github.com/alexedwards/scs/v2"
)

type AuthApplication struct {
	Queries models.Querier
	Session *scs.SessionManager
}

var Auth = AuthApplication{}

func NewAppConfig() django.AppConfig {
	var app = apps.NewAppConfig("auth")

	app.Middlewares = []http_.Middleware{
		http_.NewMiddleware(AddUserMiddleware()),
	}
	app.Init = func(settings django.Settings) error {

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

	return app
}
