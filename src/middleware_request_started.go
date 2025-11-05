package django

import (
	"errors"
	"net/http"

	core "github.com/Nigel2392/go-django/src/core"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/mux"
)

// ErrServeCanceled is an error that can be returned from a signal to indicate that the serve should be cancelled.
//
// It can be used to hijack the response and return a custom response.
//
// This signal will be sent before most middleware has been executed.
const ErrServeCanceled errs.Error = "Serve cancelled, signal hijacked response"

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
			if errors.Is(err, ErrServeCanceled) {
				return
			}
		}

		signal.H.ServeHTTP(signal.W, signal.R)

		core.SIGNAL_AFTER_REQUEST.Send(signal)
	})
}
