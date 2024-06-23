package admin

import (
	"net/http"

	"github.com/Nigel2392/django/contrib/admin/components/menu"
	"github.com/Nigel2392/django/forms/media"
	"github.com/Nigel2392/goldcrest"
)

const (
	RegisterMenuItemHook       = "admin:register_menu_item"
	RegisterFooterMenuItemHook = "admin:register_footer_menu_item"
	RegisterGlobalMedia        = "admin:register_global_media"
)

type (
	RegisterMenuItemHookFunc       func(adminSite *AdminApplication, items menu.Items)
	RegisterFooterMenuItemHookFunc func(r *http.Request, adminSite *AdminApplication, items menu.Items)
	RegisterScriptHookFunc         func(adminSite *AdminApplication) media.Media
)

func RegisterMedia(fn RegisterScriptHookFunc) {
	goldcrest.Register(RegisterGlobalMedia, 0, fn)
}
