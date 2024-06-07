package auth

import (
	"net/http"

	"github.com/Nigel2392/django"
	"github.com/Nigel2392/django/core/except"
)

func authRequiredHook(w http.ResponseWriter, r *http.Request, app *django.Application, serverError except.ServerError) {
	var (
		statusCode = serverError.StatusCode()
		_          *authenticationError
		ok         bool
	)

	if _, ok = serverError.(*authenticationError); !ok && statusCode != http.StatusUnauthorized {
		return
	}

	var redirectURL, err = app.Mux.Reverse("auth:login")
	if err != nil {
		return
	}

	http.Redirect(w, r, redirectURL, http.StatusFound)
}
