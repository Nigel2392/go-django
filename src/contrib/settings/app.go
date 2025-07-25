package settings

import (
	"context"
	"fmt"
	"net/http"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/models"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/contrib/admin/components"
	"github.com/Nigel2392/go-django/src/contrib/admin/components/menu"
	"github.com/Nigel2392/go-django/src/contrib/pages"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/permissions"
	"github.com/Nigel2392/go-django/src/views/list"
	"github.com/Nigel2392/goldcrest"
	"github.com/elliotchance/orderedmap/v2"

	_ "unsafe"
)

//go:linkname addNextUrl github.com/Nigel2392/go-django/src/contrib/pages.addNextUrl
func addNextUrl(current string, next string) string

type SettingsMenuHookFunc = func() []menu.MenuItem

type SettingsApp struct {
	*apps.AppConfig
	settings *orderedmap.OrderedMap[string, Setting]
}

const SettingsMenuHook = "settings:register_to_menu"

var AppConfig *SettingsApp = &SettingsApp{
	AppConfig: apps.NewAppConfig("settings"),
}

func NewAppConfig() django.AppConfig {

	AppConfig.Deps = []string{
		"admin",
		"pages",
	}

	AppConfig.Init = func(settings django.Settings) error {

		goldcrest.Register(
			admin.RegisterMenuItemHook, 0,
			admin.RegisterMenuItemHookFunc(func(r *http.Request, adminSite *admin.AdminApplication, items components.Items[menu.MenuItem]) {

				var menuItems = make([]menu.MenuItem, 0)
				var h = goldcrest.Get[SettingsMenuHookFunc](SettingsMenuHook)
				for _, f := range h {
					if f == nil {
						continue
					}

					var items = f()
					menuItems = append(menuItems, items...)
				}

				items.Append(&menu.SubmenuItem{
					BaseItem: menu.BaseItem{
						ItemName: "settings",
						Label:    trans.T(r.Context(), "Settings"),
					},
					Menu: &menu.Menu{
						Items: menuItems,
					},
				})
			}),
		)

		admin.RegisterApp(
			"settings",
			admin.AppOptions{
				AppLabel:            trans.S("Settings"),
				RegisterToAdminMenu: true,
				FullAdminMenu:       true,
				MenuLabel:           trans.S("Settings"),
				MenuIcon: func() string {
					return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-gear-wide-connected" viewBox="0 0 16 16">
  <path d="M7.068.727c.243-.97 1.62-.97 1.864 0l.071.286a.96.96 0 0 0 1.622.434l.205-.211c.695-.719 1.888-.03 1.613.931l-.08.284a.96.96 0 0 0 1.187 1.187l.283-.081c.96-.275 1.65.918.931 1.613l-.211.205a.96.96 0 0 0 .434 1.622l.286.071c.97.243.97 1.62 0 1.864l-.286.071a.96.96 0 0 0-.434 1.622l.211.205c.719.695.03 1.888-.931 1.613l-.284-.08a.96.96 0 0 0-1.187 1.187l.081.283c.275.96-.918 1.65-1.613.931l-.205-.211a.96.96 0 0 0-1.622.434l-.071.286c-.243.97-1.62.97-1.864 0l-.071-.286a.96.96 0 0 0-1.622-.434l-.205.211c-.695.719-1.888.03-1.613-.931l.08-.284a.96.96 0 0 0-1.186-1.187l-.284.081c-.96.275-1.65-.918-.931-1.613l.211-.205a.96.96 0 0 0-.434-1.622l-.286-.071c-.97-.243-.97-1.62 0-1.864l.286-.071a.96.96 0 0 0 .434-1.622l-.211-.205c-.719-.695-.03-1.888.931-1.613l.284.08a.96.96 0 0 0 1.187-1.186l-.081-.284c-.275-.96.918-1.65 1.613-.931l.205.211a.96.96 0 0 0 1.622-.434zM12.973 8.5H8.25l-2.834 3.779A4.998 4.998 0 0 0 12.973 8.5m0-1a4.998 4.998 0 0 0-7.557-3.779l2.834 3.78zM5.048 3.967l-.087.065zm-.431.355A4.98 4.98 0 0 0 3.002 8c0 1.455.622 2.765 1.615 3.678L7.375 8zm.344 7.646.087.065z"/>
</svg>`
				},
				MenuOrder: 1000,
			},
			admin.ModelOptions{
				RegisterToAdminMenu: true,
				Name:                "Site",
				MenuLabel:           trans.S("Sites"),
				Model:               &pages.Site{},
				Labels: map[string]func(ctx context.Context) string{
					"Name":    trans.S("Site Name"),
					"Domain":  trans.S("Domain"),
					"Port":    trans.S("Port"),
					"Default": trans.S("Is Default Site"),
					"Root":    trans.S("Root Page"),
				},
				ListView: admin.ListViewOptions{
					Columns: map[string]list.ListColumn[attrs.Definer]{
						"Root": list.LinkColumn(
							trans.S("Root Page"), "Root",
							func(r *http.Request, defs attrs.Definitions, row attrs.Definer) string {
								var node = row.(*pages.Site)
								if node.Root == nil {
									return ""
								}

								var page = node.Root
								if !permissions.HasObjectPermission(r, page, "pages:edit") {
									return ""
								}

								return addNextUrl(
									django.Reverse("admin:pages:edit", page.PK),
									r.URL.String(),
								)
							},
						),
					},
					Format: map[string]func(v any) any{
						"Root": func(v any) any {
							if v == nil {
								return ""
							}

							var page = v.(*pages.PageNode)
							if page == nil {
								return ""
							}

							return fmt.Sprintf(
								"%q (%d)", page.Title, page.ID(),
							)
						},
					},
					GetQuerySet: func(adminSite *admin.AdminApplication, app *admin.AppDefinition, model *admin.ModelDefinition) *queries.QuerySet[attrs.Definer] {
						return queries.GetQuerySet[attrs.Definer](&pages.Site{}).
							WithContext(context.Background()).
							Select("*", "Root.PK", "Root.Title")
					},
				},
				AddView: admin.FormViewOptions{
					Panels: []admin.Panel{
						admin.TitlePanel(admin.FieldPanel("Name")),
						admin.MultiPanel(
							admin.FieldPanel("Domain"),
							admin.FieldPanel("Port"),
						),
						admin.FieldPanel("Default"),
						admin.FieldPanel("Root"),
					},
				},
				EditView: admin.FormViewOptions{
					Panels: []admin.Panel{
						admin.TitlePanel(admin.FieldPanel("Name")),
						admin.MultiPanel(
							admin.FieldPanel("Domain"),
							admin.FieldPanel("Port"),
						),
						admin.FieldPanel("Default"),
						admin.FieldPanel("Root"),
					},
				},
			},
		)

		return nil
	}

	AppConfig.Ready = func() error {
		var qs = queries.GetQuerySet(&pages.Site{})
		var _, err = qs.First()
		if err != nil {
			if !errors.Is(err, errors.NoRows) {
				return errors.Wrap(err, "failed to get settings row")
			}

			var settings = models.Setup(&pages.Site{
				Name:    "Default Site",
				Domain:  "localhost",
				Port:    80,
				Default: true,
			})

			return settings.Save(context.Background())
		}

		return nil
	}

	return AppConfig
}
