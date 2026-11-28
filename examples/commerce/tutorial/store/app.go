package store

import (
	"embed"
	"net/http"

	"github.com/Nigel2392/go-django/examples/commerce/tutorial/paymentproviders"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/mux/v2"
	"golang.org/x/text/currency"
)

//go:embed templates/store/*.tmpl
var templates embed.FS

var shopCurrency = currency.EUR

func NewAppConfig() *apps.DBRequiredAppConfig {
	config := apps.NewDBAppConfig("store")
	config.Deps = []string{"shop"}
	config.ModelObjects = []attrs.Definer{&Product{}, &Delivery{}}
	config.Init = initializeDemo
	config.Ready = registerSignals
	config.TemplateConfig = &tpl.Config{
		FS:      filesystem.Sub(templates, "templates"),
		Bases:   []string{"store/base.tmpl"},
		Matches: filesystem.MatchExt(".tmpl"),
	}

	registerSuppliers()
	provider := paymentproviders.RegisterDemo()
	config.Routing = func(m mux.Multiplexer) {
		store := m.Any("", nil, "store")
		store.Get("/", http.HandlerFunc(checkoutPage), "home")
		store.Post("/checkout", http.HandlerFunc(checkout), "checkout")
		store.Get("/orders/<<id>>", http.HandlerFunc(orderPage), "order")
		store.Post("/orders/<<id>>/pay", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			pay(w, r, provider)
		}), "pay")
		store.Post("/orders/<<id>>/deliveries/<<delivery>>/confirm", http.HandlerFunc(confirmDelivery), "confirm")
	}
	return config
}
