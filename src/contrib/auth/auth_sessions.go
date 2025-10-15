//go:build !testing_auth
// +build !testing_auth

package auth

import (
	"context"
	"net/http"
	"time"

	queries "github.com/Nigel2392/go-django/queries/src"
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

	session.Set(SESSION_COOKIE_NAME, u.ID)

	core.SIGNAL_USER_LOGGED_IN.Send(core.UserWithRequest{
		User: u,
		Req:  r,
	})

	u.LastLogin = time.Now()

	// Add this as a session finalizer
	// This might allow us to skip an update query, if the caller makes one.
	session.AddFinalizer(func(r *http.Request, ctx context.Context) (context.Context, error) {
		if !u.State().Changed(true) { // update was already called
			return ctx, nil
		}

		_, err = queries.GetQuerySet(&User{}).
			WithContext(ctx).
			ExplicitSave().
			Select("LastLogin").
			Filter("ID", u.ID).
			Update(u)

		return ctx, err
	})

	return u, err
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
