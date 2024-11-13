package admin

import (
	"net/http"
	"strings"

	django "github.com/Nigel2392/go-django/src"
	autherrors "github.com/Nigel2392/go-django/src/contrib/auth/auth_errors"
	"github.com/Nigel2392/mux"
)

func RedirectLoginFailedToAdmin(w http.ResponseWriter, r *http.Request, app *django.Application, authError *autherrors.AuthenticationError) (written bool) {
	var path = strings.Trim(r.URL.Path, "/")
	var pathParts = strings.Split(path, "/")
	var _, matched, _ = AdminSite.Route.Match(mux.ANY, pathParts)
	if !matched {
		return false
	}

	var redirectURL = django.Reverse("admin:login")
	http.Redirect(w, r, redirectURL, http.StatusFound)
	return true
}
