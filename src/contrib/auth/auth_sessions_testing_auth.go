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

	models "github.com/Nigel2392/go-django/src/contrib/auth/auth-models"
	django_signals "github.com/Nigel2392/go-django/src/signals"
)

func Login(r *http.Request, u *models.User) (*models.User, error) {
	django_signals.SIGNAL_USER_LOGGED_IN.Send(django_signals.UserWithRequest{
		User: u,
		Req:  r,
	})
	u.IsLoggedIn = true
	return u, nil
}

func Logout(r *http.Request) error {
	return django_signals.SIGNAL_USER_LOGGED_OUT.Send(django_signals.UserWithRequest{
		User: nil,
		Req:  r,
	})
}
