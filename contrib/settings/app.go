package settings

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/admin/components"
	"github.com/Nigel2392/go-django/contrib/admin/components/menu"
	"github.com/Nigel2392/go-django/contrib/admin/icons"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/checks"
	"github.com/Nigel2392/go-django/src/core/command"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/staticfiles"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/goldcrest"
	"github.com/a-h/templ"
	"github.com/elliotchance/orderedmap/v2"
)

type SettingsMenuHookFunc = func(*http.Request) []menu.MenuItem
type SettingsApp struct {
	*apps.AppConfig
	settings *orderedmap.OrderedMap[string, Setting]
}

const SettingsMenuHook = "settings:register_to_menu"

var AppConfig *SettingsApp = &SettingsApp{
	AppConfig: apps.NewAppConfig("settings"),
}

func NewAppConfig() django.AppConfig {

	staticfiles.AddFS(staticFS, filesystem.MatchAnd(
		filesystem.MatchPrefix("settings/"),
		filesystem.MatchOr(
			filesystem.MatchExt(".css"),
			filesystem.MatchExt(".js"),
			filesystem.MatchExt(".png"),
			filesystem.MatchExt(".jpg"),
			filesystem.MatchExt(".jpeg"),
			filesystem.MatchExt(".svg"),
			filesystem.MatchExt(".gif"),
			filesystem.MatchExt(".ico"),
		),
	))

	AppConfig.TemplateConfig = &tpl.Config{
		AppName: "settings",
		FS:      tplFS,
		Bases:   []string{},
		Matches: filesystem.MatchAnd(
			filesystem.MatchPrefix("settings/"),
			filesystem.MatchOr(
				filesystem.MatchExt(".tmpl"),
			),
		),
	}

	AppConfig.ModelObjects = []attrs.Definer{
		&RegisteredSetting{},
	}

	AppConfig.Cmd = []command.Command{
		&SyncSettingsCmd,
	}

	AppConfig.Init = func(settings django.Settings) error {

		middlewareEnabled := django.ConfigGet(
			settings,
			"contrib.settings.SettingsForRequestMiddleware",
			true,
		)

		if middlewareEnabled {
			django.Global.Mux.Use(SettingsForRequestMiddleware)
		}

		goldcrest.Register(SettingsMenuHook, 1000, SettingsMenuHookFunc(func(r *http.Request) []menu.MenuItem {
			menuLogo := templ.ComponentFunc(icons.Component("toggles.svg"))
			return []menu.MenuItem{&menu.Item{
				Label:    trans.T(r.Context(), "Settings"),
				Logo:     menuLogo,
				ItemName: "menu_settings",
				Ordering: 1000,
				Link: func() string {
					return django.Reverse("admin:settings")
				},
			}}
		}))

		goldcrest.Register(
			admin.RegisterMenuItemHook, 0,
			admin.RegisterMenuItemHookFunc(func(r *http.Request, adminSite *admin.AdminApplication, items components.Items[menu.MenuItem]) {
				var menuLogo = templ.ComponentFunc(icons.Component("toggles.svg"))
				var menuItems = make([]menu.MenuItem, 0)
				var h = goldcrest.Get[SettingsMenuHookFunc](SettingsMenuHook)
				for _, f := range h {
					if f == nil {
						continue
					}

					var items = f(r)
					menuItems = append(menuItems, items...)
				}

				if len(menuItems) == 1 {
					items.Append(menuItems[0])
					return
				}

				items.Append(&menu.SubmenuItem{
					ItemName: "db_settings",
					Label:    trans.T(r.Context(), "Settings"),
					Logo:     menuLogo,
					Menu: &menu.Menu{
						Items: menuItems,
					},
				})
			}),
		)

		return nil
	}

	AppConfig.Ready = func() error {
		reg.lock()

		settingsRoute := admin.AdminSite.Route.Get(
			"/settings",
			http.HandlerFunc(viewSettings),
			"settings",
		)

		settingsRoute.Get("/change", http.HandlerFunc(viewSettingsEdit), "edit")
		settingsRoute.Post("/change", http.HandlerFunc(viewSettingsEdit))

		return nil
	}

	return &migrator.MigratorAppConfig{
		AppConfig: AppConfig,
		MigrationFS: filesystem.Sub(
			migrationFS, "migrations/settings",
		),
	}
}

func (a *SettingsApp) Check(ctx context.Context, djangosettings django.Settings) []checks.Message {
	var (
		messages    = a.AppConfig.Check(ctx, djangosettings)
		groups, err = groupedSettings(ctx)
	)
	if err != nil {
		panic(fmt.Errorf("error while retrieving registered settings: %w", err))
	}

	for group := range groups {
		for _, setting := range group.Settings {
			if setting.instance.ID == 0 {
				messages = append(messages, checks.Criticalf(
					"contrib.settings.SyncRequired",
					"synchronisation of setting %q is required",
					setting.instance,
					"Please run 'go run ./... sync-settings'",
					setting.instance.InternalName,
				))
			}
		}
	}

	return messages
}

// SettingsForRequestMiddleware injects the settings into the request during the middleware phase.
func SettingsForRequestMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if django.IsStaticRouteRequest(r) {
			next.ServeHTTP(w, r)
			return
		}

		newR, err := RequestWithSettings(r)
		if err != nil {
			// serve anyways
			next.ServeHTTP(w, r)
			return
		}

		next.ServeHTTP(w, newR)
	})
}
