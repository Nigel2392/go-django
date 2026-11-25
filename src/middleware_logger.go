package django

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/Nigel2392/go-django/src/core/filesystem/static"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/utils/djangoapps"
	"github.com/Nigel2392/mux/v2/hosts"
)

// loggerMiddleware is a middleware that logs the request method, time taken, remote address and path of the request.
//
// It logs the request in the following format:
//
//	<method> <time taken> <remote address> <path>
//
// The message might be prefixed and / or suffixed with additional information.
//
// If the request is a static route request, it will log the request with a debug level.
func (a *Application) loggerMiddleware(next http.Handler) http.Handler {
	var logggingEnabled = djangoapps.CONFIG.Get(APPVAR_ROUTE_LOGGING_ENABLED,
		djangoapps.CONFIG.Get(APPVAR_DEBUG, true),
	)
	if !logggingEnabled {
		return next
	}

	var log = a.Log.NameSpace("HTTP")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var startTime = time.Now()

		next.ServeHTTP(w, r)

		var logLevel = logger.INF
		if static.Request(r) {
			if !djangoapps.CONFIG.Get(APPVAR_DEBUG, true) || !djangoapps.CONFIG.Get(APPVAR_STATIC_ROUTE_LOGGING_ENABLED, false) {
				return
			}
			logLevel = logger.DBG
		}

		var (
			timeTaken  = time.Since(startTime)
			remoteAddr = hosts.GetIP(
				r, djangoapps.CONFIG.Get(
					APPVAR_REQUESTS_PROXIED, false,
				),
			)
			pathBuf = new(strings.Builder)
		)

		pathBuf.WriteString(r.URL.Path)

		if r.URL.RawQuery != "" {
			pathBuf.WriteByte('?')
			pathBuf.WriteString(r.URL.RawQuery)
		}

		log.Logf(
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
