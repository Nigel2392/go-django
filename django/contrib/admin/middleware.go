package admin

import (
	"net/http"

	"github.com/Nigel2392/django/contrib/auth"
	"github.com/Nigel2392/mux"
	"github.com/Nigel2392/mux/middleware/authentication"
)

func RequiredMiddleware(next mux.Handler) mux.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var user = authentication.Retrieve(req)
		if user == nil || !user.IsAuthenticated() {
			auth.Fail(http.StatusUnauthorized, "You need to login")
		}

		if !user.IsAdmin() {
			auth.Fail(
				http.StatusForbidden,
				"You do not have permission to access this page",
			)
		}

		next.ServeHTTP(w, req)
	})
}
