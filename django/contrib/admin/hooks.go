package admin

import (
	"net/http"

	"github.com/Nigel2392/django/contrib/admin/components/menu"
)

const (
	RegisterMenuItemHook       = "admin:register_menu_item"
	RegisterFooterMenuItemHook = "admin:register_footer_menu_item"
)

type RegisterMenuItemHookFunc func(adminSite *AdminApplication, items menu.Items)
type RegisterFooterMenuItemHookFunc func(r *http.Request, adminSite *AdminApplication, items menu.Items)
