package messages

import (
	"net/http"

	"github.com/Nigel2392/mux"
)

func MessagesMiddleware(next mux.Handler) mux.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if app.initBackend == nil {
			next.ServeHTTP(w, r)
			return
		}

		var logger = app.Logger()
		backend, err := app.initBackend(r)
		if backend == nil || err != nil {
			if err != nil {
				logger.Errorf("Error initializing messages backend: %v", err)
			}

			logger.Warn("Messages backend not configured, skipping messages middleware")
			next.ServeHTTP(w, r)
			return
		}

		r = setBackend(r, backend)
		next.ServeHTTP(w, r)

		if err := backend.Finalize(w, r); err != nil {
			logger.Errorf("Error finalizing messages backend: %v", err)
		}
	})
}
