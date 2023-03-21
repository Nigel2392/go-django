package admin

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/Nigel2392/go-django/admin/internal/models"
	"github.com/Nigel2392/go-django/auth"
	"github.com/Nigel2392/go-django/core/tracer"
	"github.com/Nigel2392/router/v3"
	"github.com/Nigel2392/router/v3/request"
)

func adminRequiredMiddleware(as *AdminSite) router.Middleware {
	return func(next router.Handler) router.Handler {
		return router.HandleFunc(func(r *request.Request) {
			if !r.User.IsAuthenticated() || !r.User.IsAdmin() {
				Unauthorized(as, r, "You do not have permission to access this page.")
				return
			}
			next.ServeHTTP(r)
		})
	}
}

func defaultDataMiddleware(as *AdminSite) router.Middleware {
	return func(next router.Handler) router.Handler {
		return router.HandleFunc(func(r *request.Request) {
			defaultDataFunc(as, r)
			next.ServeHTTP(r)
		})
	}
}

func adminHandler(as *AdminSite, m *models.Model, callback func(*AdminSite, *models.Model, *request.Request)) router.HandleFunc {
	return router.HandleFunc(func(req *request.Request) {
		if !hasAdminPerms(as, req) {
			Unauthorized(as, req, "You do not have permission to access this page.")
			return
		}

		defaultDataFunc(as, req, as.Name+" - "+m.Name)
		callback(as, m, req)
	})
}

func hasAdminPerms(as *AdminSite, req *request.Request) (retOk bool) {
	defer func() {
		if !retOk {
			as.Logger.Warningf("user with ip %s does not have permission to access admin site\n", req.IP())
			if req.User == nil {
				req.User = &auth.User{}
			}
			if !req.User.IsAuthenticated() {
				req.User = &auth.User{}
			}
			var user = req.User.(*auth.User)
			var log = SimpleLog(user, LogActionUnauthorized).WithIP(req)
			log.Meta.Set("url", req.Request.URL.String())
			log.Save(as)
		}
	}()

	if req.User == nil {
		return
	}

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

	if len(as.AllowedGroups) != 0 {
		// Check if the user has a group that is allowed to access the admin site.
		if len(uP.Groups) == 0 {
			as.DB().DB().Where(uP).Preload("Groups.Permissions").First(uP)
		}
		for _, g := range as.AllowedGroups {
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
func Unauthorized(as *AdminSite, r *request.Request, msg ...string) {
	// Runtime.Caller
	var err = errors.New("Unauthorized access")
	var callInfo = tracer.TraceSafe(err, 16, 1)

	as.Logger.Debugf("Unauthorized access by: %s\n", r.User)
	for _, c := range callInfo.Trace() {
		as.Logger.Debugf("\tUnauthorized access from: %s:%d\n", c.File, c.Line)
	}

	if len(msg) > 0 {
		for _, m := range msg {
			r.Data.AddMessage("error", m)
		}
	}
	r.Redirect(
		as.registrar.URL(router.GET, "admin:unauthorized").Format(),
		http.StatusFound,
		r.Request.URL.String())
}

func hasPerms(as *AdminSite, perms ...*auth.Permission) func(h router.Handler) router.Handler {
	return func(h router.Handler) router.Handler {
		return router.HandleFunc(func(r *request.Request) {
			if r.User == nil {
				Unauthorized(as, r, "You do not have permission to access this page.")
				return
			}
			var authUser = r.User.(*auth.User)
			for _, p := range perms {
				if !authUser.HasPerms(p) {
					Unauthorized(as, r, fmt.Sprintf("You do not have permission [%s] to access this page.", p.Name))
					return
				}
			}
			h.ServeHTTP(r)
		})
	}
}
