package reports

import (
	"net/http"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/contrib/admin/components"
	"github.com/Nigel2392/go-django/src/contrib/admin/components/menu"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/goldcrest"
	"github.com/a-h/templ"
)

type ReportsMenuHookFunc = func() []menu.MenuItem

type ReportsApp struct {
	*apps.AppConfig
}

const ReportsMenuHook = "reports:register_to_menu"

var Reports *ReportsApp = &ReportsApp{
	AppConfig: apps.NewAppConfig("reports"),
}

func NewAppConfig() django.AppConfig {
	Reports.Init = func(settings django.Settings) error {

		goldcrest.Register(
			admin.RegisterMenuItemHook, 0,
			admin.RegisterMenuItemHookFunc(func(r *http.Request, adminSite *admin.AdminApplication, items components.Items[menu.MenuItem]) {

				var menuItems = make([]menu.MenuItem, 0)
				var h = goldcrest.Get[ReportsMenuHookFunc](ReportsMenuHook)
				for _, f := range h {
					if f == nil {
						continue
					}

					var items = f()
					menuItems = append(menuItems, items...)
				}

				items.Append(&menu.SubmenuItem{
					BaseItem: menu.BaseItem{
						ItemName: "reports",
						Label:    trans.S("Reports"),
						Ordering: 990,
						Logo: templ.Raw(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-journal-text" viewBox="0 0 16 16">
  <path d="M5 10.5a.5.5 0 0 1 .5-.5h2a.5.5 0 0 1 0 1h-2a.5.5 0 0 1-.5-.5m0-2a.5.5 0 0 1 .5-.5h5a.5.5 0 0 1 0 1h-5a.5.5 0 0 1-.5-.5m0-2a.5.5 0 0 1 .5-.5h5a.5.5 0 0 1 0 1h-5a.5.5 0 0 1-.5-.5m0-2a.5.5 0 0 1 .5-.5h5a.5.5 0 0 1 0 1h-5a.5.5 0 0 1-.5-.5"/>
  <path d="M3 0h10a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2v-1h1v1a1 1 0 0 0 1 1h10a1 1 0 0 0 1-1V2a1 1 0 0 0-1-1H3a1 1 0 0 0-1 1v1H1V2a2 2 0 0 1 2-2"/>
  <path d="M1 5v-.5a.5.5 0 0 1 1 0V5h.5a.5.5 0 0 1 0 1h-2a.5.5 0 0 1 0-1zm0 3v-.5a.5.5 0 0 1 1 0V8h.5a.5.5 0 0 1 0 1h-2a.5.5 0 0 1 0-1zm0 3v-.5a.5.5 0 0 1 1 0v.5h.5a.5.5 0 0 1 0 1h-2a.5.5 0 0 1 0-1z"/>
</svg>`),
					},
					Menu: &menu.Menu{
						Items: menuItems,
					},
				})
			}),
		)

		return nil
	}

	return Reports
}
