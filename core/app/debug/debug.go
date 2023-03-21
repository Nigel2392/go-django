package debug

import (
	"github.com/Nigel2392/go-django/core/tracer"
	"github.com/Nigel2392/router/v3"
	"github.com/Nigel2392/router/v3/middleware"
	"github.com/Nigel2392/router/v3/request"
)

type DatabaseSetting struct {
	KEY      string
	ENGINE   string
	NAME     string
	SSL_MODE string
	DB_USER  string // Will be displayed.
	DB_PASS  string // Will not actually be used, but is here for completeness
}

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
func StacktraceMiddleware(settings *AppSettings) router.Middleware {
	return middleware.Recoverer(func(err error, r *request.Request) {
		var stackTrace = tracer.TraceSafe(err, 6, 5)
		r.Response.Clear()
		r.WriteString("<!DOCTYPE html>\n<html>")
		r.WriteString("<head>")
		StyleBlock(r)
		r.WriteString("</head>")
		r.WriteString("<body>")
		// Render a stacktrace of the error.
		RenderStackTrace(stackTrace, r)
		// Render additional potentially useful information.
		if tracer.STACKLOGGER_UNSAFE {
			// Render the request.
			RenderRequest(r)
			if settings != nil {
				// Render some potentially relevant settings.
				RenderSettings(r, settings)
			}
		}

		r.WriteString("</body>")
		r.WriteString("</html>")
	})
}
