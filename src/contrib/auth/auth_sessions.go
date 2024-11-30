//go:build !testing_auth
// +build !testing_auth

package auth

import (
	"net/http"

	models "github.com/Nigel2392/go-django/src/contrib/auth/auth-models"
	autherrors "github.com/Nigel2392/go-django/src/contrib/auth/auth_errors"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/mux/middleware/sessions"
)

func Login(r *http.Request, u *models.User) (*models.User, error) {
	var session = sessions.Retrieve(r)
	except.Assert(session != nil, 500, "session is nil")

	var err = session.RenewToken()
	if err != nil {
		return nil, err
	}

	u.IsLoggedIn = true

	session.Set(SESSION_COOKIE_NAME, u.ID)

	SIGNAL_USER_LOGGED_IN.Send(UserWithRequest{
		User: u,
		Req:  r,
	})

	return u, nil
}

func Logout(r *http.Request) error {
	var session = sessions.Retrieve(r)
	// except.Assert(session != nil, 500, "session is nil")
	if session == nil {
		return autherrors.ErrNoSession
	}

	if err := session.Destroy(); err != nil {
		return err
	}

	return SIGNAL_USER_LOGGED_OUT.Send(UserWithRequest{
		User: nil,
		Req:  r,
	})
}
