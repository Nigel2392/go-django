package messages

import (
	"net/http"

	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/mux"
)

func MessagesMiddleware(next mux.Handler) mux.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if app.initBackend == nil {
			next.ServeHTTP(w, r)
			return
		}

		backend, err := app.initBackend(r)
		if backend == nil || err != nil {
			if err != nil {
				logger.Errorf("Error initializing messages backend: %v", err)
			}

			logger.Warn("Messages backend not configured, skipping messages middleware")
			next.ServeHTTP(w, r)
			return
		}

		logger.Debugf("Messages backend initialized at %q: %T", r.URL.String(), backend)
		r = setBackend(r, backend)
		next.ServeHTTP(w, r)

		if err := backend.Finalize(w, r); err != nil {
			logger.Errorf("Error finalizing messages backend: %v", err)
		}
	})
}
