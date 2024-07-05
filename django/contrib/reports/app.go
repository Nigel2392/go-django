package reports

import (
	"github.com/Nigel2392/django"
	"github.com/Nigel2392/django/apps"
	"github.com/Nigel2392/django/contrib/admin"
	"github.com/Nigel2392/django/contrib/admin/components"
	"github.com/Nigel2392/django/contrib/admin/components/menu"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/goldcrest"
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
			admin.RegisterMenuItemHookFunc(func(adminSite *admin.AdminApplication, items components.Items[menu.MenuItem]) {

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
						Label:    fields.S("Reports"),
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
