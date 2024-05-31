package http_

import (
	"net/http"

	"github.com/Nigel2392/django/core"
	"github.com/Nigel2392/mux"
)

func RequestSignalMiddleware(next mux.Handler) mux.Handler {
	return mux.NewHandler(func(w http.ResponseWriter, r *http.Request) {
		var signal = &core.HttpSignal{W: w, R: r, H: next}

		core.SIGNAL_BEFORE_REQUEST.Send(signal)

		next.ServeHTTP(w, r)

		core.SIGNAL_AFTER_REQUEST.Send(signal)
	})
}
