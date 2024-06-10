package django

import (
	"errors"
	"net/http"
	"time"

	core "github.com/Nigel2392/django/core"
	"github.com/Nigel2392/django/core/errs"
	"github.com/Nigel2392/django/core/logger"
	"github.com/Nigel2392/mux"
)

// loggerMiddleware is a middleware that logs the request method, time taken, remote address and path of the request.
//
// It logs the request in the following format:
//
//	<method> <time taken> <remote address> <path>
//
// The message might be prefixed and / or suffixed with additional information.
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

// colorizeTimeTaken colorizes the time taken based on the time taken.
//
// The longer the time taken, the more red the color, starting from green.
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
