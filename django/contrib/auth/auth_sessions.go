//go:build !testing_auth
// +build !testing_auth

package auth

import (
	"net/http"

	models "github.com/Nigel2392/django/contrib/auth/auth-models"
	"github.com/Nigel2392/django/core/except"
	"github.com/Nigel2392/mux/middleware/sessions"
)

func Login(r *http.Request, u *models.User) *models.User {
	var session = sessions.Retrieve(r)
	except.Assert(session != nil, 500, "session is nil")
	//
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