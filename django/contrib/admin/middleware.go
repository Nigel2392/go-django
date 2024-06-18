package admin

import (
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"

	"github.com/Nigel2392/django/contrib/auth"
	"github.com/Nigel2392/mux"
	"github.com/Nigel2392/mux/middleware/authentication"
)

// For now only used to make sure tests pass on github actions
// This will be removed when the package is properly developed and tested
// This makes sure that the authentication check is enabled only when running on github actions
var IS_GITHUB_ACTIONS = true

func init() {
	var actionsVar = os.Getenv("GITHUB_ACTIONS")
	if slices.Contains([]string{"true", "1"}, strings.ToLower(actionsVar)) {
		IS_GITHUB_ACTIONS = true
	}
}

func RequiredMiddleware(next mux.Handler) mux.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var user = authentication.Retrieve(req)

		if IS_GITHUB_ACTIONS {
			if user == nil || !user.IsAuthenticated() {
				fmt.Println("User is not authenticated")
				auth.Fail(http.StatusUnauthorized, "You need to login", req.URL.Path)
			}

			if !user.IsAdmin() {
				auth.Fail(
					http.StatusForbidden,
					"You do not have permission to access this page",
				)
			}
		}

		next.ServeHTTP(w, req)
	})
}
