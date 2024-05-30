package http_

import (
	"net/http"

	"github.com/Nigel2392/go-signals"
)

const STATIC_URL = "/static/"

type HttpSignal struct {
	W http.ResponseWriter
	R *http.Request
	H *http.Handler
}

var handlerSignalPool = signals.NewPool[HttpSignal]()

var (
	SIGNAL_DURING_REQUEST = handlerSignalPool.Get("http.during_request") // -> Send(HttpSignal)
	SIGNAL_BEFORE_REQUEST = handlerSignalPool.Get("http.before_request") // -> Send(HttpSignal)
	SIGNAL_AFTER_REQUEST  = handlerSignalPool.Get("http.after_request")  // -> Send(HttpSignal)
)
