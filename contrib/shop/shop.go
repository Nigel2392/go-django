package shop

import (
	"github.com/Nigel2392/go-django/queries/src/drivers"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/mux"
)

type ShopAppConfig struct {
	*apps.DBRequiredAppConfig

	setup bool
}

var SHOP = &ShopAppConfig{
	DBRequiredAppConfig: apps.NewDBAppConfig("shop"),
}

func NewAppConfig() django.AppConfig {
	if SHOP == nil || SHOP.setup {
		return nil
	}

	SHOP.Deps = []string{
		"session", "auth", "admin",
	}

	SHOP.Routing = func(m mux.Multiplexer) {

	}

	SHOP.Init = func(settings django.Settings, db drivers.Database) error {
		return nil
	}

	return SHOP
}
