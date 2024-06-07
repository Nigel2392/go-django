package django

import (
	"net/http"
	"time"

	"github.com/Nigel2392/django/core/logger"
	"github.com/Nigel2392/mux"
)

func (a *Application) loggerMiddleware(next mux.Handler) mux.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			remoteAddr = mux.GetIP(
				r, ConfigGet(
					a.Settings, "django.RequestProxied", false,
				),
			)
			startTime = time.Now()
			method    = r.Method
			path      = r.URL.Path
		)

		if r.URL.RawQuery != "" {
			path += "?" + r.URL.RawQuery
		}

		next.ServeHTTP(w, r)

		var timeTaken = time.Since(startTime)
		a.Log.Infof(
			"%s %s %s %s",
			logger.Colorize(
				logger.CMD_Cyan+logger.CMD_Bold,
				method,
			),
			colorizeTimeTaken(timeTaken),
			remoteAddr,
			path,
		)
	})
}

func colorizeTimeTaken(t time.Duration) string {
	switch {
	case t < time.Millisecond*150:
		return logger.Colorize(logger.CMD_Green, t)
	case t < time.Millisecond*600:
		return logger.Colorize(logger.CMD_Yellow, t)
	default:
		return logger.Colorize(logger.CMD_Red, t)
	}
}
