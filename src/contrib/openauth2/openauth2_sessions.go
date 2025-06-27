package openauth2

import (
	"net/http"

	autherrors "github.com/Nigel2392/go-django/src/contrib/auth/auth_errors"
	"github.com/Nigel2392/go-django/src/core"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/mux/middleware/sessions"
)

func Login(r *http.Request, u *User) (*User, error) {
	var session = sessions.Retrieve(r)
	except.Assert(session != nil, 500, "session is nil")

	var err = session.RenewToken()
	if err != nil {
		return nil, err
	}

	u.IsLoggedIn = true

	session.Set(USER_ID_SESSION_KEY, u.ID)

	core.SIGNAL_USER_LOGGED_IN.Send(core.UserWithRequest{
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

	return core.SIGNAL_USER_LOGGED_OUT.Send(core.UserWithRequest{
		User: nil,
		Req:  r,
	})
}
