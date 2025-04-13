package admin

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/components"
	cmpts "github.com/Nigel2392/go-django/src/contrib/admin/components"
	"github.com/Nigel2392/go-django/src/contrib/admin/components/menu"
	"github.com/Nigel2392/go-django/src/contrib/admin/icons"
	autherrors "github.com/Nigel2392/go-django/src/contrib/auth/auth_errors"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/staticfiles"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/goldcrest"
	"github.com/Nigel2392/mux"
	"github.com/a-h/templ"
	"github.com/elliotchance/orderedmap/v2"
)

var (
	//go:embed assets/**
	adminFS embed.FS

	tplFS    fs.FS
	staticFS fs.FS
)

func init() {
	var err error
	tplFS, err = fs.Sub(adminFS, "assets/templates")
	if err != nil {
		panic(err)
	}

	staticFS, err = fs.Sub(adminFS, "assets/static")
	if err != nil {
		panic(err)
	}

}

const BASE_KEY = "admin"

var AdminSite *AdminApplication = &AdminApplication{
	AppConfig: apps.NewAppConfig("admin"),
	Apps: orderedmap.NewOrderedMap[
		string, *AppDefinition,
	](),
	Ordering: make([]string, 0),
	Route: mux.NewRoute(
		mux.ANY, "admin/", nil, "admin",
	),
}

var (
	RegisterApp   = AdminSite.RegisterApp
	IsReady       = AdminSite.IsReady
	ConfigureAuth = AdminSite.configureAuth
)

func NewAppConfig() django.AppConfig {
	// AdminSite.Route.Use(RequiredMiddleware)
	var iconHTML template.HTML

	// AdminSite.Deps = []string{
	// "auth",
	// }
	AdminSite.Init = func(settings django.Settings) error {
		settings.App().Mux.AddRoute(AdminSite.Route)

		autherrors.OnAuthenticationError(RedirectLoginFailedToAdmin)

		components.Register("admin.header", cmpts.Header)

		components.Register("admin.heading", cmpts.Heading)
		components.Register("admin.heading1", cmpts.Heading1)
		components.Register("admin.heading2", cmpts.Heading2)
		components.Register("admin.heading3", cmpts.Heading3)
		components.Register("admin.heading4", cmpts.Heading4)
		components.Register("admin.heading5", cmpts.Heading5)
		components.Register("admin.heading6", cmpts.Heading6)

		components.Register("admin.button", cmpts.NewButton)
		components.Register("admin.button.primary", cmpts.ButtonPrimary)
		components.Register("admin.button.secondary", cmpts.ButtonSecondary)
		components.Register("admin.button.success", cmpts.ButtonSuccess)
		components.Register("admin.button.danger", cmpts.ButtonDanger)
		components.Register("admin.button.warning", cmpts.ButtonWarning)

		goldcrest.Register(
			RegisterFooterMenuItemHook, 0,
			RegisterFooterMenuItemHookFunc(func(r *http.Request, adminSite *AdminApplication, items cmpts.Items[menu.MenuItem]) {
				items.Append(&menu.Item{
					BaseItem: menu.BaseItem{
						ItemName: "logout",
						Label:    trans.S("Logout"),
						Logo: templ.Raw(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-box-arrow-right" viewBox="0 0 16 16">
	<!-- The MIT License (MIT) -->
	<!-- Copyright (c) 2011-2024 The Bootstrap Authors -->
  	<path fill-rule="evenodd" d="M10 12.5a.5.5 0 0 1-.5.5h-8a.5.5 0 0 1-.5-.5v-9a.5.5 0 0 1 .5-.5h8a.5.5 0 0 1 .5.5v2a.5.5 0 0 0 1 0v-2A1.5 1.5 0 0 0 9.5 2h-8A1.5 1.5 0 0 0 0 3.5v9A1.5 1.5 0 0 0 1.5 14h8a1.5 1.5 0 0 0 1.5-1.5v-2a.5.5 0 0 0-1 0z"/>
  	<path fill-rule="evenodd" d="M15.854 8.354a.5.5 0 0 0 0-.708l-3-3a.5.5 0 0 0-.708.708L14.293 7.5H5.5a.5.5 0 0 0 0 1h8.793l-2.147 2.146a.5.5 0 0 0 .708.708z"/>
</svg>`),
					},
					Link: func() string {
						return django.Reverse("admin:logout")
					},
				})
			}),
		)

		return nil
	}

	AdminSite.Ready = func() error {

		if AdminSite.auth == nil {
			assert.Fail(
				"AdminApplication.Ready: authentication was not setup with admin.ConfigureAuth(...)",
			)
		}

		if AdminSite.auth.GetLoginForm == nil && AdminSite.auth.GetLoginHandler == nil {
			assert.Fail(
				"AdminApplication.Ready: GetLoginForm or GetLoginHandler was not set with admin.ConfigureAuth(...)",
			)
		}

		if AdminSite.auth.Logout == nil {
			assert.Fail(
				"AdminApplication.Ready: logoutFunc was not set with admin.ConfigureAuth(...)",
			)
		}

		if err := icons.Register(staticFS,
			"admin/icons/view.svg",
			"admin/icons/history.svg",
			"admin/icons/no-view.svg",
		); err != nil {
			panic(err)
		}

		var replacer = strings.NewReplacer(
			"xmlns=\"http://www.w3.org/2000/svg\"", "",
			"<svg", "<symbol",
			"</svg>", "</symbol>",
		)

		var icons = icons.Icons()
		var htmlString = make([]string, 0, len(icons))
		for _, icon := range icons {
			htmlString = append(htmlString, replacer.Replace(
				string(icon.HTML()),
			))
		}
		iconHTML = template.HTML(
			fmt.Sprintf(
				`<svg xmlns="http://www.w3.org/2000/svg" style="display: none;"><defs>%s</defs></svg>`,
				strings.Join(htmlString, ""),
			),
		)
		// First initialize routes which do not require authentication
		AdminSite.Route.Get(
			"login/", mux.NewHandler(loginHandler),
			"login", // admin:login
		)
		AdminSite.Route.Post(
			"login/", mux.NewHandler(loginHandler),
			"login", // admin:login
		)

		AdminSite.Route.Get(
			"logout/", mux.NewHandler(logoutHandler),
			"logout", // admin:logout
		)

		AdminSite.Route.Get(
			"relogin/", mux.NewHandler(reloginHandler),
			"relogin", // admin:relogin
		)

		// Add authentication/administrator middleware to all subsequent routes added
		AdminSite.Route.Use(
			RequiredMiddleware,
		)

		AdminSite.Route.Get(
			"", views.Serve(HomeHandler), "home",
		)

		// Initialize authenticated routes
		var baseApps = AdminSite.Route.Handle(
			mux.ANY, "apps/<<app_name>>/",
			newAppHandler(AppHandler),
			"apps", // admin:apps
		)

		var baseModelsRoute = baseApps.Handle(
			mux.ANY, "model/<<model_name>>/",
			newModelHandler(ModelListHandler),
			"model", // admin:apps:model
		)

		baseModelsRoute.Handle(
			mux.ANY, "add/",
			newModelHandler(ModelAddHandler),
			"add", // admin:apps:model:add
		)

		baseModelsRoute.Handle(
			mux.ANY, "edit/<<model_id>>/",
			newInstanceHandler(ModelEditHandler),
			"edit", // admin:apps:model:edit
		)

		baseModelsRoute.Handle(
			mux.ANY, "delete/<<model_id>>/",
			newInstanceHandler(ModelDeleteHandler),
			"delete", // admin:apps:model:delete
		)

		// External / Extension URLs root
		var routeExtensions = AdminSite.Route.Handle(
			mux.ANY, "ext/", nil,
			"ext", // admin:ext
		)

		// Register all custom app URLs to the extension route
		for front := AdminSite.Apps.Front(); front != nil; front = front.Next() {
			var app = front.Value
			if app.Routing != nil {
				app.Routing(routeExtensions)
			}
			//for _, url := range app.URLs {
			//	url.Register(routeExtensions)
			//}
		}

		for front := AdminSite.Apps.Front(); front != nil; front = front.Next() {
			front.Value.OnReady(AdminSite)
		}

		// Mark the admin site as ready
		AdminSite.ready.Store(true)

		return nil
	}

	staticfiles.AddFS(staticFS, filesystem.MatchAnd(
		filesystem.MatchPrefix("admin/"),
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

	AdminSite.TemplateConfig = &tpl.Config{
		AppName: "admin",
		FS:      tplFS,
		Bases: []string{
			"admin/skeleton.tmpl",
			"admin/base.tmpl",
		},
		Matches: filesystem.MatchAnd(
			filesystem.MatchPrefix("admin/"),
			filesystem.MatchOr(
				filesystem.MatchExt(".tmpl"),
			),
		),
		Funcs: template.FuncMap{
			"icons": func() template.HTML {
				return iconHTML
			},
			"icon": func(name string) template.HTML {
				return template.HTML(fmt.Sprintf(`<svg class="icon %s">
	<use href="#%s"></use>
</svg>`, name, name))
			},
			"menu": func(r *http.Request) template.HTML {
				var m = &menu.Menu{}
				var menuItems = cmpts.NewItems[menu.MenuItem]()
				var hooks = goldcrest.Get[RegisterMenuItemHookFunc](RegisterMenuItemHook)
				for _, hook := range hooks {
					hook(AdminSite, menuItems)
				}
				m.Items = menuItems.All()
				var buf = new(bytes.Buffer)
				m.Component().Render(r.Context(), buf)
				return template.HTML(buf.String())
			},
			"script_hook_output": func() media.Media {
				var hooks = goldcrest.Get[RegisterMediaHookFunc](RegisterGlobalMediaHook)
				var m media.Media = media.NewMedia()
				for _, hook := range hooks {
					var hook_m = hook(AdminSite)
					if hook_m != nil {
						m = m.Merge(hook_m)
					}
				}
				return m
			},
			"footer_menu": func(r *http.Request) template.HTML {
				var m = &menu.Menu{}
				var menuItems = cmpts.NewItems[menu.MenuItem]()
				var hooks = goldcrest.Get[RegisterFooterMenuItemHookFunc](RegisterFooterMenuItemHook)
				for _, hook := range hooks {
					hook(r, AdminSite, menuItems)
				}
				m.Items = menuItems.All()
				var buf = new(bytes.Buffer)
				m.Component().Render(r.Context(), buf)
				return template.HTML(buf.String())
			},
		},
	}

	return AdminSite
}

func newHandler(handler func(w http.ResponseWriter, r *http.Request)) mux.Handler {
	return mux.NewHandler(func(w http.ResponseWriter, req *http.Request) {

		handler(w, req)
	})
}

func newInstanceHandler(handler func(w http.ResponseWriter, req *http.Request, adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition, instance attrs.Definer)) mux.Handler {
	return newHandler(func(w http.ResponseWriter, req *http.Request) {
		var (
			vars      = mux.Vars(req)
			appName   = vars.Get("app_name")
			modelName = vars.Get("model_name")
			modelID   = vars.Get("model_id")
		)

		if modelName == "" || appName == "" || modelID == "" {
			except.Fail(http.StatusBadRequest, "App, Model name and Model ID is required")
			return
		}

		var app, ok = AdminSite.Apps.Get(appName)
		if !ok {
			except.Fail(http.StatusBadRequest, "App not found")
			return
		}

		model, ok := app.Models.Get(modelName)
		if !ok {
			except.Fail(http.StatusBadRequest, "Model not found")
			return
		}

		var instance, err = model.GetInstance(modelID)
		if err != nil {
			except.Fail(
				http.StatusInternalServerError,
				"Failed to get instance: %s", err,
			)
			return
		}

		handler(w, req, AdminSite, app, model, instance)
	})
}

func newModelHandler(handler func(w http.ResponseWriter, r *http.Request, adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition)) mux.Handler {
	return newHandler(func(w http.ResponseWriter, req *http.Request) {
		var (
			vars      = mux.Vars(req)
			appName   = vars.Get("app_name")
			modelName = vars.Get("model_name")
		)

		if modelName == "" || appName == "" {
			except.Fail(http.StatusBadRequest, "App and Model name is required")
			return
		}

		var app, ok = AdminSite.Apps.Get(appName)
		if !ok {
			except.Fail(http.StatusBadRequest, "App not found")
			return
		}

		model, ok := app.Models.Get(modelName)
		if !ok {
			except.Fail(http.StatusBadRequest, "Model not found")
			return
		}

		handler(w, req, AdminSite, app, model)
	})
}

func newAppHandler(handler func(w http.ResponseWriter, r *http.Request, adminSite *AdminApplication, app *AppDefinition)) mux.Handler {
	return newHandler(func(w http.ResponseWriter, req *http.Request) {
		var vars = mux.Vars(req)
		var appName = vars.Get("app_name")

		if appName == "" {
			except.Fail(http.StatusBadRequest, "App name is required")
			return
		}

		var app, ok = AdminSite.Apps.Get(appName)
		if !ok {
			except.Fail(http.StatusBadRequest, "App not found")
			return
		}

		handler(w, req, AdminSite, app)
	})
}
