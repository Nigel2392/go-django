package django

import (
	"net/http"

	"github.com/Nigel2392/django/core/except"
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
	HOOK_SERVER_ERROR = "django.ServerError"
	HOOK_SETUP_NOSURF = "django.SetupNosurf"
)
