//go:build testing_auth
// +build testing_auth

/*
Package auth provides a way to login and logout users.

This file is used in place of auth_sessions.go when testing.

This is due to sessions not being available in testing.

Logging in and out should only have an effect on the session (Aside for settings IsLoggedIn)
thus making the default file safe to override.
*/

package auth

import (
	"net/http"

	"github.com/Nigel2392/go-django/src/core"
)

func Login(r *http.Request, u *User) (*User, error) {
	core.SIGNAL_USER_LOGGED_IN.Send(core.UserWithRequest{
		User: u,
		Req:  r,
	})
	u.IsLoggedIn = true
	return u, nil
}

func Logout(r *http.Request) error {
	return core.SIGNAL_USER_LOGGED_OUT.Send(core.UserWithRequest{
		User: nil,
		Req:  r,
	})
}
