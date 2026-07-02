package django

import (
	"net/http"

	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/justinas/nosurf"
)

type markedResponseWriter struct {
	http.ResponseWriter
	wasWritten bool
}

func (w *markedResponseWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}

func (w *markedResponseWriter) WriteHeader(statusCode int) {
	w.wasWritten = true
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *markedResponseWriter) Write(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}

	w.wasWritten = true
	return w.ResponseWriter.Write(data)
}

type (
	DjangoHook      func(*Application) error
	ServerErrorHook func(w http.ResponseWriter, r *http.Request, app *Application, err except.ServerError)
	NosurfSetupHook func(app *Application, handler *nosurf.CSRFHandler)
)

const (
	HOOK_SERVER_ERROR    = "django.ServerError" // ran when an exception is caught with err of type [except.ServerError]
	HOOK_SERVER_SHUTDOWN = "django.ServerQuit"  // ran after django's http(s) server(s) shutdown.
	HOOK_SETUP_NOSURF    = "django.SetupNosurf" // ran during http server startup, allows for changing CSRF handler options
)
