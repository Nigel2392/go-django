package attrs

import "github.com/Nigel2392/go-signals"

var (
	modelSignalPool = signals.NewPool[Definer]()

	OnModelRegister = modelSignalPool.Get("attrs.OnRegister")
)
