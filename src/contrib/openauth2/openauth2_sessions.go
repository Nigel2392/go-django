package openauth2

import (
	"net/http"

	autherrors "github.com/Nigel2392/go-django/src/contrib/auth/auth_errors"
	openauth2models "github.com/Nigel2392/go-django/src/contrib/openauth2/openauth2_models"
	"github.com/Nigel2392/go-django/src/core/except"
	django_signals "github.com/Nigel2392/go-django/src/signals"
	"github.com/Nigel2392/mux/middleware/sessions"
)

func Login(r *http.Request, u *openauth2models.User) (*openauth2models.User, error) {
	var session = sessions.Retrieve(r)
	except.Assert(session != nil, 500, "session is nil")

	var err = session.RenewToken()
	if err != nil {
		return nil, err
	}

	u.IsLoggedIn = true

	session.Set(USER_ID_SESSION_KEY, u.ID)

	django_signals.SIGNAL_USER_LOGGED_IN.Send(django_signals.UserWithRequest{
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

	return django_signals.SIGNAL_USER_LOGGED_OUT.Send(django_signals.UserWithRequest{
		User: nil,
		Req:  r,
	})
}
