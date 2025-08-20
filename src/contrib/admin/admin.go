package admin

import (
	"bytes"
	"context"
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
	"github.com/Nigel2392/go-django/src/core/checks"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/staticfiles"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/go-django/src/models"
	"github.com/Nigel2392/go-django/src/permissions"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/goldcrest"
	"github.com/Nigel2392/mux"
	"github.com/Nigel2392/mux/middleware/authentication"
	"github.com/a-h/templ"
	"github.com/elliotchance/orderedmap/v2"
)

type ModelHandlerFunc func(func(w http.ResponseWriter, req *http.Request, adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition)) mux.Handler

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

	AdminSite.Deps = []string{
		"messages",
	}
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

				var menuItem = &menu.Menu{
					Items: make([]menu.MenuItem, 0),
				}

				var user = authentication.Retrieve(r)
				var model = FindDefinition(user)

				if user.IsAdmin() && permissions.HasObjectPermission(r, user, "admin:edit") {
					menuItem.Items = append(menuItem.Items, &menu.Item{
						BaseItem: menu.BaseItem{
							ItemName: "user_change",
							Label:    trans.T(r.Context(), "Edit Account"),
							Logo: templ.Raw(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-person-gear" viewBox="0 0 16 16">
								<!-- The MIT License (MIT) -->
								<!-- Copyright (c) 2011-2024 The Bootstrap Authors -->
								<path d="M11 5a3 3 0 1 1-6 0 3 3 0 0 1 6 0M8 7a2 2 0 1 0 0-4 2 2 0 0 0 0 4m.256 7a4.5 4.5 0 0 1-.229-1.004H3c.001-.246.154-.986.832-1.664C4.484 10.68 5.711 10 8 10q.39 0 .74.025c.226-.341.496-.65.804-.918Q8.844 9.002 8 9c-5 0-6 3-6 4s1 1 1 1zm3.63-4.54c.18-.613 1.048-.613 1.229 0l.043.148a.64.64 0 0 0 .921.382l.136-.074c.561-.306 1.175.308.87.869l-.075.136a.64.64 0 0 0 .382.92l.149.045c.612.18.612 1.048 0 1.229l-.15.043a.64.64 0 0 0-.38.921l.074.136c.305.561-.309 1.175-.87.87l-.136-.075a.64.64 0 0 0-.92.382l-.045.149c-.18.612-1.048.612-1.229 0l-.043-.15a.64.64 0 0 0-.921-.38l-.136.074c-.561.305-1.175-.309-.87-.87l.075-.136a.64.64 0 0 0-.382-.92l-.148-.045c-.613-.18-.613-1.048 0-1.229l.148-.043a.64.64 0 0 0 .382-.921l-.074-.136c-.306-.561.308-1.175.869-.87l.136.075a.64.64 0 0 0 .92-.382zM14 12.5a1.5 1.5 0 1 0-3 0 1.5 1.5 0 0 0 3 0"/>
							</svg>`),
						},
						Link: func() string {
							return django.Reverse(
								"admin:apps:model:edit",
								model.App().Name, model.GetName(), attrs.PrimaryKey(user.(attrs.Definer)),
							)
						},
					})
				}

				menuItem.Items = append(menuItem.Items, &menu.Item{
					BaseItem: menu.BaseItem{
						ItemName: "logout",
						Label:    trans.T(r.Context(), "Logout"),
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

				items.Append(&menu.DropdownItem{
					BaseItem: menu.BaseItem{
						ItemName: "account_details",
						Label:    trans.T(r.Context(), "Account Details"),
						Logo: templ.Raw(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-person" viewBox="0 0 16 16">
							<!-- The MIT License (MIT) -->
							<!-- Copyright (c) 2011-2024 The Bootstrap Authors -->
							<path d="M8 8a3 3 0 1 0 0-6 3 3 0 0 0 0 6m2-3a2 2 0 1 1-4 0 2 2 0 0 1 4 0m4 8c0 1-1 1-1 1H3s-1 0-1-1 1-4 6-4 6 3 6 4m-1-.004c-.001-.246-.154-.986-.832-1.664C11.516 10.68 10.289 10 8 10s-3.516.68-4.168 1.332c-.678.678-.83 1.418-.832 1.664z"/>
						</svg>`),
					},
					Menu: menuItem,
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

		AdminSite.Route.Use(func(next mux.Handler) mux.Handler {
			return mux.NewHandler(func(w http.ResponseWriter, r *http.Request) {
				var vars = mux.Vars(r)
				var appNameSlice = vars["app_name"]
				var appName string
				if len(appNameSlice) == 0 || appNameSlice[0] == "" {
					appName = "admin"
				} else {
					appName = appNameSlice[0]
				}

				var djangoApp, ok = django.Global.Apps.Get(appName)
				if !ok {
					logger.Errorf(
						"AdminSite.Route.Use: app %q not found in django.Global.Apps, falling back to AdminSite",
						appName,
					)
					djangoApp = AdminSite
				}

				next.ServeHTTP(w, r.WithContext(django.ContextWithApp(
					r.Context(), djangoApp,
				)))
			})
		})

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
			mux.ANY, "<<model_name>>/",
			NewModelHandler("app_name", "model_name", ModelListHandler),
			"model", // admin:apps:model
		)

		baseModelsRoute.Handle(
			mux.ANY, "add/",
			NewModelHandler("app_name", "model_name", ModelAddHandler),
			"add", // admin:apps:model:add
		)

		baseModelsRoute.Handle(
			mux.ANY, "edit/<<model_id>>/",
			NewInstanceHandler("app_name", "model_name", "model_id", ModelEditHandler),
			"edit", // admin:apps:model:edit
		)

		baseModelsRoute.Handle(
			mux.ANY, "delete/<<model_id>>/",
			NewInstanceHandler("app_name", "model_name", "model_id", ModelDeleteHandler),
			"delete", // admin:apps:model:delete
		)

		baseModelsRoute.Handle(
			mux.ANY, "delete/",
			NewModelHandler("app_name", "model_name", ModelBulkDeleteHandler),
			"bulk_delete", // admin:apps:model:delete
		)

		var hooks = goldcrest.Get[RegisterModelsRouteHookFunc](AdminModelHookRegisterRoute)
		for _, hook := range hooks {
			hook(AdminSite, baseModelsRoute)
		}

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
			"admin/messages.tmpl",
			"admin/base.tmpl",
		},
		Matches: filesystem.MatchAnd(
			filesystem.MatchPrefix("admin/"),
			filesystem.MatchOr(
				filesystem.MatchExt(".tmpl"),
			),
		),
		Funcs: template.FuncMap{
			"string": attrs.ToString,
			"icons": func() template.HTML {
				return iconHTML
			},
			"icon": func(name string, attrs ...string) template.HTML {
				var attr = strings.Join(attrs, " ")
				return template.HTML(fmt.Sprintf(`<svg class="icon %s" %s>
	<use href="#%s"></use>
</svg>`, name, attr, name))
			},
			"menu": func(r *http.Request) template.HTML {
				var m = &menu.Menu{}
				var menuItems = cmpts.NewItems[menu.MenuItem]()
				var hooks = goldcrest.Get[RegisterMenuItemHookFunc](RegisterMenuItemHook)
				for _, hook := range hooks {
					hook(r, AdminSite, menuItems)
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

func (a *AdminApplication) Check(ctx context.Context, settings django.Settings) []checks.Message {
	var messages = a.AppConfig.Check(ctx, settings)

	for head := a.Apps.Front(); head != nil; head = head.Next() {

		var app = head.Value
		if !nameRegex.MatchString(app.Name) {
			messages = append(messages, checks.Criticalf(
				"admin.app_name_invalid",
				"App name does not match regex %v", app.Name,
				"",
				nameRegex,
			))
		}

		var _, ok = django.Global.Apps.Get(app.Name)
		if !ok {
			messages = append(messages, checks.Criticalf(
				"admin.app_not_registered",
				"App %q is not registered in django.Global.Apps",
				nil, "",
				app.Name,
			))
		}

		for front := app.Models.Front(); front != nil; front = front.Next() {
			var implementsSave, implementsDelete = models.ImplementsMethods(front.Value.Model)
			if !implementsSave || !implementsDelete {
				//logger.Warnf(
				//	"Model %q is not fully implemented, canSave: %t, canDelete: %t",
				//	front.Value.GetName(), implementsSave, implementsDelete,
				//)
				messages = append(messages, checks.Warningf(
					"admin.model_not_fully_implemented",
					"Model %q is not fully implemented, canSave: %t, canDelete: %t",
					front.Value.Model,
					"Implement both models.ContextSaver and models.ContextDeleter interfaces to"+
						" avoid issues with saving or deleting instances of this model.",
					front.Value.GetName(), implementsSave, implementsDelete,
				))
			}

			messages = append(
				messages,
				front.Value.Check(ctx, app)...,
			)
		}
	}
	return messages
}

func newHandler(handler func(w http.ResponseWriter, r *http.Request)) mux.Handler {
	return mux.NewHandler(handler)
}

func NewInstanceHandler(appnameVar, modelVar, idVar string, handler func(w http.ResponseWriter, req *http.Request, adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition, instance attrs.Definer)) mux.Handler {
	return newHandler(func(w http.ResponseWriter, req *http.Request) {
		var (
			vars      = mux.Vars(req)
			appName   = vars.Get(appnameVar)
			modelName = vars.Get(modelVar)
			modelID   = vars.Get(idVar)
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

		model, ok := app.modelsByName[modelName]
		if !ok {
			except.Fail(http.StatusBadRequest, "Model not found")
			return
		}

		var instance, err = model.GetInstance(req.Context(), modelID)
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

func NewModelHandler(appnameVar, modelVar string, handler func(w http.ResponseWriter, r *http.Request, adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition), fail ...func(w http.ResponseWriter, r *http.Request, msg string)) mux.Handler {
	if len(fail) == 0 {
		fail = append(fail, func(w http.ResponseWriter, r *http.Request, msg string) {
			Home(w, r, msg)
		})
	}

	if len(fail) > 1 {
		assert.Fail("NewModelHandler: too many fail functions provided, only one is allowed")
		return nil
	}

	return newHandler(func(w http.ResponseWriter, req *http.Request) {
		var (
			vars      = mux.Vars(req)
			appName   = vars.Get(appnameVar)
			modelName = vars.Get(modelVar)
		)

		if modelName == "" || appName == "" {
			fail[0](w, req, "App and Model name is required")
			return
		}

		var app, ok = AdminSite.Apps.Get(appName)
		if !ok {
			fail[0](w, req, "App not found")
			return
		}

		model, ok := app.modelsByName[modelName]
		if !ok {
			fail[0](w, req, "Model not found")
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
