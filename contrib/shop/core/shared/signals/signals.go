package signals

import (
	"net/http"

	"github.com/Nigel2392/go-signals"
)

type BaseSignal struct {
	Request *http.Request
}

type Signal[T any] = signals.Signal[T]

type Changed[TYP any] struct {
	Old TYP
	New TYP
}

type ModelSignals[SIGNAL_T any] struct {
	Created signals.Signal[SIGNAL_T]
	Updated signals.Signal[SIGNAL_T]
	Deleted signals.Signal[SIGNAL_T]
}

var Pool = signals.NewGPool()
