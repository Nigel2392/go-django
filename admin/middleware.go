package admin

import (
	"github.com/Nigel2392/router/v3"
	"github.com/Nigel2392/router/v3/request"
)

func adminSiteMiddleware(next router.Handler) router.Handler {
	return router.HandleFunc(func(r *request.Request) {
		if !r.User.IsAuthenticated() || !r.User.IsAdmin() {
			Unauthorized(r, "You do not have permission to access this page.")
			return
		}
		if !r.User.HasPermissions(PermissionViewAdminSite) {
			Unauthorized(r, "You do not have permission to access this page.")
			return
		}

		var apps = adminSite_Apps.InOrder()
		r.Data.Set("apps", apps)

		next.ServeHTTP(r)
	})
}
