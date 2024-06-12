package admin

import (
	"bytes"
	"embed"
	"html/template"
	"io/fs"
	"net/http"

	"github.com/Nigel2392/django"
	"github.com/Nigel2392/django/apps"
	"github.com/Nigel2392/django/components"
	cmpts "github.com/Nigel2392/django/contrib/admin/components"
	"github.com/Nigel2392/django/contrib/admin/components/menu"
	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/core/except"
	"github.com/Nigel2392/django/core/staticfiles"
	"github.com/Nigel2392/django/core/tpl"
	"github.com/Nigel2392/django/views"
	"github.com/Nigel2392/goldcrest"
	"github.com/Nigel2392/mux"
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
	RegisterApp = AdminSite.RegisterApp
	IsReady     = AdminSite.IsReady
)

func NewAppConfig() django.AppConfig {
	// AdminSite.Route.Use(RequiredMiddleware)

	AdminSite.Init = func(settings django.Settings) error {
		settings.App().Mux.AddRoute(AdminSite.Route)

		staticfiles.AddFS(staticFS, tpl.MatchAnd(
			tpl.MatchPrefix("admin/"),
			tpl.MatchOr(
				tpl.MatchExt(".css"),
				tpl.MatchExt(".js"),
				tpl.MatchExt(".png"),
				tpl.MatchExt(".jpg"),
				tpl.MatchExt(".jpeg"),
				tpl.MatchExt(".svg"),
				tpl.MatchExt(".gif"),
				tpl.MatchExt(".ico"),
			),
		))

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

		tpl.Add(tpl.Config{
			AppName: "admin",
			FS:      tplFS,
			Bases: []string{
				"admin/skeleton.tmpl",
				"admin/base.tmpl",
			},
			Matches: tpl.MatchAnd(
				tpl.MatchPrefix("admin/"),
				tpl.MatchOr(
					tpl.MatchExt(".tmpl"),
				),
			),
			Funcs: template.FuncMap{
				"menu": func(r *http.Request) template.HTML {
					var m = &menu.Menu{}
					var menuItems = menu.NewItems()
					var hooks = goldcrest.Get[RegisterMenuItemHookFunc](RegisterMenuItemHook)
					for _, hook := range hooks {
						hook(AdminSite, menuItems)
					}
					m.Items = menuItems.All()
					var buf = new(bytes.Buffer)
					m.Component().Render(r.Context(), buf)
					return template.HTML(buf.String())
				},
			},
		})

		return nil
	}

	AdminSite.Ready = func() error {

		// First initialize routes which do not require authentication
		AdminSite.Route.Get(
			"login/", views.Serve(LoginHandler),
			"login", // admin:login
		)
		AdminSite.Route.Post(
			"login/", views.Serve(LoginHandler),
			"login", // admin:login
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
			for _, url := range app.URLs {
				url.Register(routeExtensions)
			}
		}

		for front := AdminSite.Apps.Front(); front != nil; front = front.Next() {
			front.Value.OnReady(AdminSite)
		}

		// Mark the admin site as ready
		AdminSite.ready.Store(true)

		return nil
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
			http.Error(w, "App, Model name and Model ID is required", http.StatusBadRequest)
			return
		}

		var app, ok = AdminSite.Apps.Get(appName)
		if !ok {
			http.Error(w, "App not found", http.StatusNotFound)
			return
		}

		model, ok := app.Models.Get(modelName)
		if !ok {
			http.Error(w, "Model not found", http.StatusNotFound)
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
			http.Error(w, "App and Model name is required", http.StatusBadRequest)
			return
		}

		var app, ok = AdminSite.Apps.Get(appName)
		if !ok {
			http.Error(w, "App not found", http.StatusNotFound)
			return
		}

		model, ok := app.Models.Get(modelName)
		if !ok {
			http.Error(w, "Model not found", http.StatusNotFound)
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
			http.Error(w, "App name is required", http.StatusBadRequest)
			return
		}

		var app, ok = AdminSite.Apps.Get(appName)
		if !ok {
			http.Error(w, "App not found", http.StatusNotFound)
			return
		}

		handler(w, req, AdminSite, app)
	})
}
