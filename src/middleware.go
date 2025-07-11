package django

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	core "github.com/Nigel2392/go-django/src/core"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/mux"
)

type staticRouteKey struct{}

var DEFAULT_LOGGING_ENABLED = true
var method_color = []string{logger.CMD_Cyan, logger.CMD_Bold}

func IsStaticRoute(r *http.Request) bool {
	var isStaticRoute = r.Context().Value(staticRouteKey{})
	if isStaticRoute, ok := isStaticRoute.(*bool); ok {
		return *isStaticRoute
	}
	return false
}

func MarkStatic(r *http.Request, isStatic bool) *http.Request {
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

func MarkStaticRouteMiddleware(next mux.Handler) mux.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, MarkStatic(r, true))
	})
}

func NonStaticMiddleware(middleware mux.Middleware) mux.Middleware {
	return func(next mux.Handler) mux.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if IsStaticRoute(r) {
				next.ServeHTTP(w, r)
				return
			}

			var handler = middleware(next)
			handler.ServeHTTP(w, r)
		})
	}
}

// loggerMiddleware is a middleware that logs the request method, time taken, remote address and path of the request.
//
// It logs the request in the following format:
//
//	<method> <time taken> <remote address> <path>
//
// The message might be prefixed and / or suffixed with additional information.
func (a *Application) loggerMiddleware(next mux.Handler) mux.Handler {
	var logggingEnabled = ConfigGet(a.Settings, APPVAR_ROUTE_LOGGING_ENABLED,
		ConfigGet(a.Settings, APPVAR_DEBUG, true),
	)
	if !logggingEnabled {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var startTime = time.Now()

		next.ServeHTTP(w, r)

		var logLevel = logger.INF
		if IsStaticRoute(r) {
			logLevel = logger.DBG
			// return
		}

		var (
			timeTaken  = time.Since(startTime)
			remoteAddr = mux.GetIP(
				r, ConfigGet(
					a.Settings, APPVAR_REQUESTS_PROXIED, false,
				),
			)
			pathBuf = new(strings.Builder)
		)

		pathBuf.WriteString(r.URL.Path)

		if r.URL.RawQuery != "" {
			pathBuf.WriteByte('?')
			pathBuf.WriteString(r.URL.RawQuery)
		}

		a.Log.Logf(
			logLevel,
			"%s %s %s %s",
			logger.Colorize(
				r.Context(),
				method_color,
				r.Method,
			),
			colorizeTimeTaken(r.Context(), timeTaken),
			remoteAddr,
			pathBuf.String(),
		)
	})
}

// colorizeTimeTaken colorizes the time taken based on the time taken.
//
// The longer the time taken, the more red the color, starting from green.
func colorizeTimeTaken(ctx context.Context, t time.Duration) string {
	switch {
	case t < time.Millisecond*150:
		return logger.Colorize(ctx, logger.CMD_Green, t)
	case t < time.Millisecond*600:
		return logger.Colorize(ctx, logger.CMD_Yellow, t)
	default:
		return logger.Colorize(ctx, logger.CMD_Red, t)
	}
}

// CancelServeError is an error that can be returned from a signal to indicate that the serve should be cancelled.
//
// It can be used to hijack the response and return a custom response.
//
// This signal will be sent before most middleware has been executed.
const CancelServeError errs.Error = "Serve cancelled, signal hijacked response"

// RequestSignalMiddleware is a middleware that sends signals before and after a request is served.
//
// It sends SIGNAL_BEFORE_REQUEST before the request is served and SIGNAL_AFTER_REQUEST after the request is served.
//
// The signal it sends is of type *core.HttpSignal.
//
// This can be used to initialize and / or clean up resources before and after a request is served.
func RequestSignalMiddleware(next mux.Handler) mux.Handler {
	return mux.NewHandler(func(w http.ResponseWriter, r *http.Request) {
		var signal = &core.HttpSignal{W: w, R: r, H: next}

		if err := core.SIGNAL_BEFORE_REQUEST.Send(signal); err != nil {
			if errors.Is(err, CancelServeError) {
				return
			}
		}

		next.ServeHTTP(w, r)

		core.SIGNAL_AFTER_REQUEST.Send(signal)
	})
}
