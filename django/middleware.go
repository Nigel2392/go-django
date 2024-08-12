package django

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	core "github.com/Nigel2392/django/core"
	"github.com/Nigel2392/django/core/errs"
	"github.com/Nigel2392/django/core/logger"
	"github.com/Nigel2392/mux"
)

type loggingEnabledKey string

var DEFAULT_LOGGING_ENABLED = true
var method_color = []string{logger.CMD_Cyan, logger.CMD_Bold}
var logKey loggingEnabledKey = "logging"

func LoggingEnabled(r *http.Request) bool {
	var enabled = r.Context().Value(logKey)
	if enabled == nil {
		return DEFAULT_LOGGING_ENABLED
	}
	if enabled, ok := enabled.(*bool); ok {
		return *enabled
	}
	return DEFAULT_LOGGING_ENABLED
}

func LogRequest(r *http.Request, enabled bool) *http.Request {
	var ctx = r.Context()
	if v := ctx.Value(logKey); v != nil {
		if v, ok := v.(*bool); ok {
			*v = enabled
		}
		return r
	}

	ctx = context.WithValue(ctx, logKey, &enabled)
	return r.WithContext(ctx)
}

func LoggingDisabledMiddleware(next mux.Handler) mux.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, LogRequest(r, false))
	})
}

// loggerMiddleware is a middleware that logs the request method, time taken, remote address and path of the request.
//
// It logs the request in the following format:
//
//	<method> <time taken> <remote address> <path>
//
// The message might be prefixed and / or suffixed with additional information.
func (a *Application) loggerMiddleware(next mux.Handler) mux.Handler {
	var logggingEnabled = ConfigGet(a.Settings, "LOGGING_ENABLED",
		ConfigGet(a.Settings, APPVAR_DEBUG, true),
	)
	if !logggingEnabled {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var startTime = time.Now()

		r = LogRequest(r, true)

		next.ServeHTTP(w, r)

		var logLevel = logger.INF
		if !LoggingEnabled(r) {
			logLevel = logger.DBG
		}

		var (
			timeTaken  = time.Since(startTime)
			remoteAddr = mux.GetIP(
				r, ConfigGet(
					a.Settings, "django.RequestProxied", false,
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
				method_color,
				r.Method,
			),
			colorizeTimeTaken(timeTaken),
			remoteAddr,
			pathBuf.String(),
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
