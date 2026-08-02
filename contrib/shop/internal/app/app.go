package app

import (
	"fmt"
	"net/http"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/admin/components"
	"github.com/Nigel2392/go-django/contrib/admin/components/menu"
	"github.com/Nigel2392/go-django/contrib/shop/util/signals"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/goldcrest"
	"github.com/Nigel2392/mux"
	"github.com/a-h/templ"
)

type HttpShopHandler interface {
	ServeHTTP(http.ResponseWriter, *http.Request, *ShopAppConfig)
}

func ShopToHttpHandler(shop *ShopAppConfig) func(func(http.ResponseWriter, *http.Request, *ShopAppConfig)) http.Handler {
	return func(f func(http.ResponseWriter, *http.Request, *ShopAppConfig)) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			f(w, r, shop)
		})
	}
}

type shopHandlerFunc func(w http.ResponseWriter, r *http.Request, shop *ShopAppConfig)

func (s shopHandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request, shop *ShopAppConfig) {
	s(w, r, shop)
}

type ShopAppConfig struct {
	*apps.DBRequiredAppConfig
	SIGNALS     signals.Manager
	ADMIN_ROUTE *mux.Route
	ADMIN_MENU  func(r *http.Request) []menu.MenuItem
}

func (a *ShopAppConfig) BuildRouting(m mux.Multiplexer) {
	if a.ADMIN_ROUTE == nil {
		panic(fmt.Sprintf("admin route not set for %T", a))
	}

	a.Routing(a.ADMIN_ROUTE)
}

func (a *ShopAppConfig) OnReady() error {
	if err := a.DBRequiredAppConfig.OnReady(); err != nil {
		return err
	}

	goldcrest.Register(
		admin.RegisterMenuItemHook, 0,
		admin.RegisterMenuItemHookFunc(func(r *http.Request, adminSite *admin.AdminApplication, items components.Items[menu.MenuItem]) {
			items.Append(&menu.SubmenuItem{
				BaseItem: menu.BaseItem{
					ItemName: "shop",
					Label:    trans.T(r.Context(), "Commerce"),
					Ordering: 6,
					Logo: templ.Raw(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-cart4" viewBox="0 0 16 16">
  	<path d="M0 2.5A.5.5 0 0 1 .5 2H2a.5.5 0 0 1 .485.379L2.89 4H14.5a.5.5 0 0 1 .485.621l-1.5 6A.5.5 0 0 1 13 11H4a.5.5 0 0 1-.485-.379L1.61 3H.5a.5.5 0 0 1-.5-.5M3.14 5l.5 2H5V5zM6 5v2h2V5zm3 0v2h2V5zm3 0v2h1.36l.5-2zm1.11 3H12v2h.61zM11 8H9v2h2zM8 8H6v2h2zM5 8H3.89l.5 2H5zm0 5a1 1 0 1 0 0 2 1 1 0 0 0 0-2m-2 1a2 2 0 1 1 4 0 2 2 0 0 1-4 0m9-1a1 1 0 1 0 0 2 1 1 0 0 0 0-2m-2 1a2 2 0 1 1 4 0 2 2 0 0 1-4 0"/>
</svg>`),
				},
				Menu: &menu.Menu{
					Items: a.ADMIN_MENU(r),
				},
			})
		}),
	)

	return nil
}
