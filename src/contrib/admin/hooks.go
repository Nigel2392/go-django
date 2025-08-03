package admin

import (
	"fmt"
	"net/http"

	"github.com/Nigel2392/go-django/src/contrib/admin/components"
	"github.com/Nigel2392/go-django/src/contrib/admin/components/menu"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/goldcrest"
	"github.com/Nigel2392/mux"
)

const (
	// RegisterAppMenuItemHook    = "admin:register_menu_item:<app_name>"
	RegisterMenuItemHook       = "admin:register_menu_item"
	RegisterFooterMenuItemHook = "admin:register_footer_menu_item"
	RegisterGlobalMediaHook    = "admin:register_global_media"
	RegisterNavBreadCrumbHook  = "admin:register_breadcrumb"
	RegisterNavActionHook      = "admin:register_nav_action"

	RegisterHomePageBreadcrumbHook = "admin:home:register_breadcrumb"
	RegisterHomePageActionHook     = "admin:home:register_action"
	RegisterHomePageComponentHook  = "admin:home:register_component"

	AdminModelHookAdd           = "admin:model:add"
	AdminModelHookEdit          = "admin:model:edit"
	AdminModelHookDelete        = "admin:model:delete"
	AdminModelHookRegisterRoute = "admin:model:register_route"
)

var (
	// Register a custom component for an app registered to the admin.
	// The app name is used to identify the app.
	// It should be formatted as "admin:<app_name>:register_admin_page_component"
	// This will then return a string which can be used to register the component to the app.
	RegisterAdminPageComponentHook = func(appname string) string {
		return fmt.Sprintf("admin:%s:register_component", appname)
	}
)

type (
	RegisterMenuItemHookFunc       = func(r *http.Request, adminSite *AdminApplication, items components.Items[menu.MenuItem])
	RegisterAppMenuItemHookFunc    func(r *http.Request, adminSite *AdminApplication, app *AppDefinition) []menu.MenuItem
	RegisterFooterMenuItemHookFunc = func(r *http.Request, adminSite *AdminApplication, items components.Items[menu.MenuItem])
	RegisterMediaHookFunc          = func(adminSite *AdminApplication) media.Media
	RegisterBreadCrumbHookFunc     = func(r *http.Request, adminSite *AdminApplication) []BreadCrumb
	RegisterNavActionHookFunc      = func(r *http.Request, adminSite *AdminApplication) []Action

	RegisterHomePageBreadcrumbHookFunc = func(*http.Request, *AdminApplication, []BreadCrumb)
	RegisterHomePageActionHookFunc     = func(*http.Request, *AdminApplication, []Action)
	RegisterHomePageComponentHookFunc  = func(*http.Request, *AdminApplication) AdminPageComponent

	RegisterAdminAppPageComponentHookFunc = func(r *http.Request, adminSite *AdminApplication, app *AppDefinition) AdminPageComponent

	AdminModelHookFunc          = func(r *http.Request, adminSite *AdminApplication, model *ModelDefinition, instance attrs.Definer)
	RegisterModelsRouteHookFunc = func(adminSite *AdminApplication, route mux.Multiplexer, newHandler func(func(w http.ResponseWriter, req *http.Request, adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition)) mux.Handler)
)

// Register an item to the django admin menu (sidebar).
func RegisterGlobalMenuItem(fn RegisterMenuItemHookFunc) {
	goldcrest.Register(RegisterMenuItemHook, 0, fn)
}

// Register an item to the django admin menu (sidebar) for a specific app.
func RegisterAppMenuItem(appName string, fn RegisterAppMenuItemHookFunc) {
	var b = fmt.Sprintf("%s:%s", RegisterMenuItemHook, appName)
	goldcrest.Register(b, 0, fn)
}

// Register an item to the django admin menu's footer.
func RegisterGlobalFooterMenuItem(fn RegisterFooterMenuItemHookFunc) {
	goldcrest.Register(RegisterFooterMenuItemHook, 0, fn)
}

// Register a media hook to the admin site.
//
// This is used to add custom CSS & JS to the admin site.
func RegisterGlobalMedia(fn RegisterMediaHookFunc) {
	goldcrest.Register(RegisterGlobalMediaHook, 0, fn)
}

// Register a breadcrumb to the admin site.
func RegisterGlobalNavBreadCrumb(fn RegisterBreadCrumbHookFunc) {
	goldcrest.Register(RegisterNavBreadCrumbHook, 0, fn)
}

// Register a navigation action to the admin site.
func RegisterGlobalNavAction(fn RegisterNavActionHookFunc) {
	goldcrest.Register(RegisterNavActionHook, 0, fn)
}

// Register a breadcrumb for the home page.
func RegisterHomePageBreadcrumb(f RegisterHomePageBreadcrumbHookFunc) {
	goldcrest.Register(RegisterHomePageBreadcrumbHook, 0, f)
}

// Register an action item for the home page.
func RegisterHomePageAction(f RegisterHomePageActionHookFunc) {
	goldcrest.Register(RegisterHomePageActionHook, 0, f)
}

// Register a custom component for the home page.
func RegisterHomePageComponent(f RegisterHomePageComponentHookFunc) {
	goldcrest.Register(RegisterHomePageComponentHook, 0, f)
}

// Register a hook to be called when a model is added.
func RegisterModelAddHook(f AdminModelHookFunc) {
	goldcrest.Register(AdminModelHookAdd, 0, f)
}

// Register a hook to be called when a model is edited.
func RegisterModelEditHook(f AdminModelHookFunc) {
	goldcrest.Register(AdminModelHookEdit, 0, f)
}

// Register a hook to be called when a model is deleted.
func RegisterModelDeleteHook(f AdminModelHookFunc) {
	goldcrest.Register(AdminModelHookDelete, 0, f)
}

// Register a hook to register a route for models.
func RegisterModelsRouteHook(f RegisterModelsRouteHookFunc) {
	goldcrest.Register(AdminModelHookRegisterRoute, 0, f)
}

// Register a custom component for an app registered to the admin.
func RegisterAdminAppPageComponent(appname string, f RegisterAdminAppPageComponentHookFunc) {
	goldcrest.Register(
		RegisterAdminPageComponentHook(appname), 0, f,
	)
}
