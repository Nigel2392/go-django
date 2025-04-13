package admin

import (
	"net/http"

	django "github.com/Nigel2392/go-django/src"
	autherrors "github.com/Nigel2392/go-django/src/contrib/auth/auth_errors"
	"github.com/Nigel2392/go-django/src/permissions"
	"github.com/Nigel2392/mux"
	"github.com/Nigel2392/mux/middleware/authentication"
)

func RequiredMiddleware(next mux.Handler) mux.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var user = authentication.Retrieve(req)

		if user == nil || !user.IsAuthenticated() {
			autherrors.Fail(http.StatusUnauthorized, "You need to login", req.URL.Path)
		}

		if !user.IsAdmin() || !permissions.HasPermission(req, "admin:access_admin") {
			http.Redirect(
				w, req,
				django.Reverse("admin:relogin"),
				http.StatusSeeOther,
			)
			return
		}

		next.ServeHTTP(w, req)
	})
}
