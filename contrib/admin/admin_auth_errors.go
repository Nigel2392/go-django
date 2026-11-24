package admin

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	autherrors "github.com/Nigel2392/go-django/contrib/auth/auth_errors"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/mux/v2"
)

func RedirectLoginFailedToAdmin(w http.ResponseWriter, r *http.Request, app *django.Application, authError *autherrors.AuthenticationError) (written bool) {
	var path = strings.Trim(r.URL.Path, "/")
	var _, matched, _ = AdminSite.Route.Match(mux.ANY, path)
	if !matched {
		return false
	}

	var redirectURL = django.Reverse("admin:login")
	var nextURL = authError.NextURL
	if nextURL != "" {
		redirectURL = fmt.Sprintf(
			"%s?next=%s",
			redirectURL,
			url.QueryEscape(nextURL),
		)
	}

	http.Redirect(w, r, redirectURL, http.StatusFound)
	return true
}
