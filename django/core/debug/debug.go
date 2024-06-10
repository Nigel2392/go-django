package debug

import (
	"net/http"
	"strings"

	"github.com/Nigel2392/django/core/debug/tracer"
	"github.com/Nigel2392/mux"
	"github.com/Nigel2392/mux/middleware"
)

// Some database settings that are useful to display in the stacktrace.
//
// The DB_PASS field is not actually used, but is here for completeness.
//
// It will be displayed in the stacktrace as "********".
type DatabaseSetting struct {
	KEY      string
	ENGINE   string
	NAME     string
	SSL_MODE string
	DB_USER  string // Will be displayed.
	DB_PASS  string // Will not actually be used, but is here for completeness
}

// Some application settings that are useful to display in the stacktrace.
type AppSettings struct {
	DEBUG  bool
	HOST   string
	PORT   int
	ROUTES string

	DATABASES []DatabaseSetting
}

// This is a middleware function that should be used in the router.
//
// It will recover from a panic and render a stacktrace of the error.
//
// You can limit the information that is displayed by setting the
// tracer.STACKLOGGER_UNSAFE variable to false.
//
// Or when using the default app.Applicationm you can set the
// app.Application.DEBUG variable to false. (This will disable it! Recommended.)
func StacktraceMiddleware(settings *AppSettings) mux.Middleware {
	return middleware.Recoverer(func(err error, w http.ResponseWriter, r *http.Request) {
		var stackTrace = tracer.TraceSafe(err, 8, 5)
		var buf = new(strings.Builder)
		buf.WriteString("<!DOCTYPE html>\n<html>")
		buf.WriteString("<head>")
		StyleBlock(buf)
		buf.WriteString("</head>")
		buf.WriteString("<body>")
		// Render standard error message and debug disable warning.
		RenderStdInfo(buf, stackTrace)
		// Render a stacktrace of the error.
		RenderStackTrace(stackTrace, buf, r)
		// Render additional potentially useful information.
		if tracer.STACKLOGGER_UNSAFE {
			// Render the request.
			RenderRequest(buf, r)
			if settings != nil {
				// Render some potentially relevant settings.
				RenderSettings(buf, settings)
			}
		}

		buf.WriteString("</body>")
		buf.WriteString("</html>")
	})
}
