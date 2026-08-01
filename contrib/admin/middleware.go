package admin

import (
	"net/http"

	autherrors "github.com/Nigel2392/go-django/contrib/auth/auth_errors"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/permissions"
	"github.com/Nigel2392/mux"
	"github.com/Nigel2392/mux/middleware/authentication"
)

func AppMiddleware(next mux.Handler) mux.Handler {
	return mux.NewHandler(func(w http.ResponseWriter, r *http.Request) {
		var vars = mux.Vars(r)
		var appNameSlice = vars["app_name"]
		var appName string
		if len(appNameSlice) == 0 || appNameSlice[0] == "" {
			appName = "admin"
		} else {
			appName = appNameSlice[0]
		}

		var djangoApp, ok = django.Global.Apps.Get(appName)
		if !ok {
			logger.Errorf(
				"AdminSite.Route.Use: app %q not found in django.Global.Apps, falling back to AdminSite",
				appName,
			)
			djangoApp = AdminSite
		}

		next.ServeHTTP(w, r.WithContext(django.ContextWithApp(
			r.Context(), djangoApp,
		)))
	})
}

func RequiredMiddleware(next mux.Handler) mux.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var user = authentication.Retrieve(req)

		if user == nil || !user.IsAuthenticated() {
			autherrors.Fail(http.StatusUnauthorized, trans.T(req.Context(), "You need to login"), req.URL.Path)
		}

		if user.IsAdmin() {
			goto serveAdmin
		}

		if !permissions.HasPermission(req, "admin:access_admin") {
			logger.Warnf(
				"User \"%s\" tried to access admin without permission",
				attrs.ToString(user),
			)
			ReLogin(w, req, req.URL.Path)
			return
		}

	serveAdmin:
		next.ServeHTTP(w, req)
	})
}
