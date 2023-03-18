package admin

import (
	"fmt"
	"net/http"

	"github.com/Nigel2392/go-django/admin/internal/models"
	"github.com/Nigel2392/go-django/auth"
	"github.com/Nigel2392/go-django/logger"
	"github.com/Nigel2392/router/v3"
	"github.com/Nigel2392/router/v3/request"
)

func adminRequiredMiddleware(next router.Handler) router.Handler {
	return router.HandleFunc(func(r *request.Request) {
		if !r.User.IsAuthenticated() {
			Unauthorized(r, "You do not have permission to access this page.")
			return
		}
		next.ServeHTTP(r)
	})
}

func defaultDataMiddleware(next router.Handler) router.Handler {
	return router.HandleFunc(func(r *request.Request) {
		defaultDataFunc(r)
		next.ServeHTTP(r)
	})
}

func adminHandler(m *models.Model, callback func(*models.Model, *request.Request)) router.HandleFunc {
	return router.HandleFunc(func(req *request.Request) {
		if !hasAdminPerms(req) {
			Unauthorized(req, "You do not have permission to access this page.")
			return
		}

		defaultDataFunc(req, AdminSite_Name+" - "+m.Name)
		callback(m, req)
	})
}

func hasAdminPerms(req *request.Request) (retOk bool) {
	defer func() {
		if !retOk {
			AdminSite_Logger.Warningf("user with ip %s does not have permission to access admin site\n", req.IP())
			var user = req.User.(*auth.User)
			var log = SimpleLog(user, LogActionUnauthorized).WithIP(req)
			log.Meta.Set("url", req.Request.URL.String())
			log.Save()
		}
	}()

	if !req.User.IsAuthenticated() {
		return
	}

	if req.User.IsAdmin() {
		retOk = true
		return
	}

	var uP, ok = req.User.(*auth.User)
	if ok {
		if !uP.HasPerms(PermissionViewAdminSite) {
			retOk = false
			return
		}
	}

	if len(AdminSite_AllowedGroups) != 0 {
		// Check if the user has a group that is allowed to access the admin site.
		if len(uP.Groups) == 0 {
			admin_db.Where(uP).Preload("Groups.Permissions").First(uP)
		}
		for _, g := range AdminSite_AllowedGroups {
			if !uP.HasGroup(g...) {
				retOk = false
				return
			}
		}
	}
	retOk = true
	return
}

// Unauthorized redirects the user to the unauthorized page.
// It will also log a stacktrace of the code that called this function.
func Unauthorized(r *request.Request, msg ...string) {
	// Runtime.Caller
	var callInfo = logger.GetCaller(0)

	AdminSite_Logger.Debugf("Unauthorized access by: %s\n", r.User)
	for _, c := range callInfo {
		AdminSite_Logger.Debugf("\tUnauthorized access from: %s:%d\n", c.File, c.Line)
	}

	if len(msg) > 0 {
		for _, m := range msg {
			r.Data.AddMessage("error", m)
		}
	}
	r.Redirect(
		adminSite.URL(router.GET, "admin:unauthorized").Format(),
		http.StatusFound,
		r.Request.URL.String())
}

func hasPerms(perms ...*auth.Permission) func(h router.Handler) router.Handler {
	return func(h router.Handler) router.Handler {
		return router.HandleFunc(func(r *request.Request) {
			if r.User == nil {
				Unauthorized(r, "You do not have permission to access this page.")
				return
			}
			var authUser = r.User.(*auth.User)
			for _, p := range perms {
				if !authUser.HasPerms(p) {
					Unauthorized(r, fmt.Sprintf("You do not have permission [%s] to access this page.", p.Name))
					return
				}
			}
			h.ServeHTTP(r)
		})
	}
}
