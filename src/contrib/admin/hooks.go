package admin

import (
	"net/http"

	"github.com/Nigel2392/go-django/src/contrib/admin/components"
	"github.com/Nigel2392/go-django/src/contrib/admin/components/menu"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/goldcrest"
)

const (
	RegisterMenuItemHook       = "admin:register_menu_item"
	RegisterFooterMenuItemHook = "admin:register_footer_menu_item"
	RegisterGlobalMediaHook    = "admin:register_global_media"
	RegisterNavBreadCrumbHook  = "admin:register_breadcrumb"
	RegisterNavActionHook      = "admin:register_nav_action"

	RegisterHomePageBreadcrumbHook = "admin:home:register_breadcrumb"
	RegisterHomePageActionHook     = "admin:home:register_action"
	RegisterHomePageComponentHook  = "admin:home:register_component"

	AdminModelHookAdd    = "admin:model:add"
	AdminModelHookEdit   = "admin:model:edit"
	AdminModelHookDelete = "admin:model:delete"
)

type (
	RegisterMenuItemHookFunc       func(adminSite *AdminApplication, items components.Items[menu.MenuItem])
	RegisterFooterMenuItemHookFunc func(r *http.Request, adminSite *AdminApplication, items components.Items[menu.MenuItem])
	RegisterMediaHookFunc          func(adminSite *AdminApplication) media.Media
	RegisterBreadCrumbHookFunc     func(r *http.Request, adminSite *AdminApplication) []BreadCrumb
	RegisterNavActionHookFunc      func(r *http.Request, adminSite *AdminApplication) []Action

	RegisterHomePageBreadcrumbHookFunc = func(*http.Request, *AdminApplication, []BreadCrumb)
	RegisterHomePageActionHookFunc     = func(*http.Request, *AdminApplication, []Action)
	RegisterHomePageComponentHookFunc  = func(*http.Request, *AdminApplication) HomePageComponent

	AdminModelHookFunc func(r *http.Request, adminSite *AdminApplication, model *ModelDefinition, instance attrs.Definer)
)

// Register an item to the django admin menu (sidebar).
func RegisterGlobalMenuItem(fn RegisterMenuItemHookFunc) {
	goldcrest.Register(RegisterMenuItemHook, 0, fn)
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
