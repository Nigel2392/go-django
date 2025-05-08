package attrs

import "github.com/Nigel2392/go-signals"

var (
	modelSignalPool = signals.NewPool[Definer]()

	OnBeforeModelRegister = modelSignalPool.Get("attrs.OnBeforeRegister")
	OnModelRegister       = modelSignalPool.Get("attrs.OnRegister")
)
