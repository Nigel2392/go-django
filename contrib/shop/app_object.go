package shop

import (
	"fmt"

	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/mux"
)

type ShopAppConfig struct {
	*apps.DBRequiredAppConfig
	SIGNALS     SignalManager
	ADMIN_ROUTE *mux.Route
}

func (a *ShopAppConfig) BuildRouting(m mux.Multiplexer) {
	if a.ADMIN_ROUTE == nil {
		panic(fmt.Sprintf("admin route not set for %T", a))
	}

	a.Routing(a.ADMIN_ROUTE)
}
