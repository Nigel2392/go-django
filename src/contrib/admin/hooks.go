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
	AdminModelHookFunc             func(r *http.Request, adminSite *AdminApplication, model *ModelDefinition, instance attrs.Definer)
)

func RegisterGlobalMenuItem(fn RegisterMenuItemHookFunc) {
	goldcrest.Register(RegisterMenuItemHook, 0, fn)
}

func RegisterGlobalFooterMenuItem(fn RegisterFooterMenuItemHookFunc) {
	goldcrest.Register(RegisterFooterMenuItemHook, 0, fn)
}

func RegisterGlobalMedia(fn RegisterMediaHookFunc) {
	goldcrest.Register(RegisterGlobalMediaHook, 0, fn)
}

func RegisterGlobalNavBreadCrumb(fn RegisterBreadCrumbHookFunc) {
	goldcrest.Register(RegisterNavBreadCrumbHook, 0, fn)
}

func RegisterGlobalNavAction(fn RegisterNavActionHookFunc) {
	goldcrest.Register(RegisterNavActionHook, 0, fn)
}
