package admin

import "github.com/Nigel2392/django/contrib/admin/components/menu"

const (
	RegisterMenuItemHook = "admin:register_menu_item"
)

type RegisterMenuItemHookFunc func(adminSite *AdminApplication, items menu.Items)
