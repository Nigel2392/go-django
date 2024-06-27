package auth

import (
	"net/http"
	"net/url"

	"github.com/Nigel2392/django"
	"github.com/Nigel2392/django/core/except"
)

// isAuthErrorHook is a hook that will redirect the user to the login page if an authentication error occurs.
//
// Authentication errors can be raised using auth.Fail(...)
//
// Under the hood this induced a panic; which is then caught by django and allows for more advanced error handling.
//
// This makes sure boilerplate code for failing auth is not repeated.
//
// It also allows for a more consistent way to handle auth errors.
func isAuthErrorHook(w http.ResponseWriter, r *http.Request, app *django.Application, serverError except.ServerError) {
	var (
		authError *authenticationError
		ok        bool
	)

	if authError, ok = serverError.(*authenticationError); !ok {
		return
	}

	var redirectURL, err = app.Mux.Reverse("auth:login")
	if err != nil {
		return
	}

	if authError.NextURL != "" {
		var u, err = url.Parse(redirectURL)
		if err != nil {
			goto respond
		}

		q := u.Query()
		q.Set("next", authError.NextURL)
		u.RawQuery = q.Encode()
		redirectURL = u.String()
	}

respond:
	http.Redirect(w, r, redirectURL, http.StatusFound)
}
