package admin

import (
	"fmt"
	"net/http"
	"net/url"

	django "github.com/Nigel2392/go-django/src"
	autherrors "github.com/Nigel2392/go-django/src/contrib/auth/auth_errors"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/logger"
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

		if user.IsAdmin() {
			goto serveAdmin
		}

		if !permissions.HasPermission(req, "admin:access_admin") {
			logger.Warnf(
				"User \"%s\" tried to access admin without permission",
				attrs.ToString(user),
			)
			var nextURL = req.URL.Path
			var redirectURL = fmt.Sprintf(
				"%s?next=%s",
				django.Reverse("admin:relogin"),
				url.QueryEscape(nextURL),
			)
			http.Redirect(
				w, req,
				redirectURL,
				http.StatusSeeOther,
			)
			return
		}

	serveAdmin:
		next.ServeHTTP(w, req)
	})
}
