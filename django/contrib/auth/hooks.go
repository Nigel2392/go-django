package auth

import (
	"net/http"
	"net/url"

	"github.com/Nigel2392/django"
	"github.com/Nigel2392/django/core/except"
)

func authRequiredHook(w http.ResponseWriter, r *http.Request, app *django.Application, serverError except.ServerError) {
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
