package core

import (
	"net/http"

	"github.com/Nigel2392/go-signals"
	"github.com/Nigel2392/mux"
)

type HttpSignal struct {
	W http.ResponseWriter
	R *http.Request
	H mux.Handler
}

var handlerSignalPool = signals.NewPool[*HttpSignal]()

var (
	SIGNAL_BEFORE_REQUEST = handlerSignalPool.Get("http.before_request") // -> Send(HttpSignal)
	SIGNAL_AFTER_REQUEST  = handlerSignalPool.Get("http.after_request")  // -> Send(HttpSignal)
)
