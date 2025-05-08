package core

import "github.com/Nigel2392/go-signals"

var core_signalPool = signals.NewPool[any]()

var (
	OnDjangoReady = core_signalPool.Get("django.OnReady")       // -> Send()
	OnModelsReady = core_signalPool.Get("django.OnModelsReady") // -> Send()
)
