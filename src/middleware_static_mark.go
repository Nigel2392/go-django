package django

import (
	"context"
	"net/http"

	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/mux"
)

var DEFAULT_LOGGING_ENABLED = true
var method_color = []string{logger.CMD_Cyan, logger.CMD_Bold}

type staticRouteKey struct{}

// IsStaticRouteRequest checks if the request is marked as a static route request.
//
// It checks the request context for a value of type `*bool` with the key [staticRouteKey]{}.
func IsStaticRouteRequest(r *http.Request) bool {
	var isStaticRoute = r.Context().Value(staticRouteKey{})
	if isStaticRoute, ok := isStaticRoute.(*bool); ok && isStaticRoute != nil {
		return *isStaticRoute
	}
	return false
}

// RequestWithStaticMark returns a new request with the static route mark set to the given value.
//
// It does so by adding a value to the request context using [http.Request.WithContext].
//
// The value is stored as a pointer to a bool - this is done to ensure
// that middleware that runs after this can modify the value if needed.
func RequestWithStaticMark(r *http.Request, isStatic bool) *http.Request {
	var ctx = r.Context()
	if v := ctx.Value(staticRouteKey{}); v != nil {
		if v, ok := v.(*bool); ok {
			*v = isStatic
		}
		return r
	}

	ctx = context.WithValue(ctx, staticRouteKey{}, &isStatic)
	return r.WithContext(ctx)
}

// MarkStaticRouteMiddleware marks the request as a static route request.
//
// This is used to skip certain middleware for static routes, such as the logger middleware.
//
// It should be used in conjunction with [IsStaticRouteRequest] to check if the current route is marked as a static route.
//
// This should be used for route handlers, not as a global mux middleware.
// This middleware should be executed before any other middleware that needs to check if the route is static,
// which can be done by adding the middleware using [mux.Route.Preprocess].
func MarkStaticRouteMiddleware(next mux.Handler) mux.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, RequestWithStaticMark(r, true))
	})
}

// NonStaticMiddleware is a middleware that skips the middleware if the request is a static route request.
//
// This is useful for middleware that should not be executed for static routes, such as the logger middleware.
func NonStaticMiddleware(middleware mux.Middleware) mux.Middleware {
	return func(next mux.Handler) mux.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if IsStaticRouteRequest(r) {
				next.ServeHTTP(w, r)
				return
			}

			var handler = middleware(next)
			handler.ServeHTTP(w, r)
		})
	}
}
