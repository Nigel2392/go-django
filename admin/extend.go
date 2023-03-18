package admin

import (
	"fmt"

	"github.com/Nigel2392/go-django/admin/internal/menu"
	"github.com/Nigel2392/go-django/auth"
	"github.com/Nigel2392/go-django/core/httputils"
	"github.com/Nigel2392/go-django/core/modelutils/namer"
	"github.com/Nigel2392/router/v3"
	"github.com/Nigel2392/router/v3/templates/extensions"
)

// Internal models are models that are used by the admin site to manage
//
// extra pages, such as the logs page.
type internalModel struct {
	model     interface{}
	viewFunc  router.HandleFunc
	perms     auth.PermissionMap
	registrar router.Registrar
}

// The internal application menu item.
//
// This will only be loaded once.
//
// This is where models such as the admin logs will be registered.
var __internalApp *menu.Item

// The extensions application menu item.
//
// This will only be loaded once.
//
// This is where all the extensions will be registered.
var __extensionsApp *menu.Item

// The internal menu.
//
// Will only be loaded once.
func internal_menu_items() *menu.Item {
	if __internalApp != nil {
		return __internalApp
	}

	__internalApp = menu.NewItem(
		InternalAppName,
		fmt.Sprintf("%s/%s", AdminSite_URL, InternalAppName),
		fmt.Sprintf("/%s", InternalAppName),
		0,
	)

	return __internalApp
}

// The extensions menu.
//
// Will only be loaded once.
func extensions_menu_items() *menu.Item {
	if __extensionsApp != nil {
		return __extensionsApp
	}

	__extensionsApp = menu.NewItem(
		ExtensionsAppName,
		httputils.NicePath(false, AdminSite_URL, EXTENSION_URL),
		httputils.NicePath(false, EXTENSION_URL),
		0,
	)

	return __extensionsApp
}

// Register an internal model.
//
// This will register the model to the internal menu.
func register_internal_model(m *internalModel) {
	if __internalApp == nil {
		internal_menu_items()
	}

	if m.perms == nil {
		m.perms = auth.PermissionMap{
			"view": auth.NewPermission("view", m.model),
			"list": auth.NewPermission("list", m.model),
		}
	}

	var modelName = namer.GetModelName(m.model)
	__internalApp.Children(menu.NewItem(
		httputils.TitleCaser.String(modelName),
		fmt.Sprintf("%s/%s", __internalApp.URL.WholeURL, httputils.SimpleSlugify(modelName)),
		fmt.Sprintf("/%s", httputils.SimpleSlugify(modelName)),
		0,
		m,
	))
}

// Register an extension.
//
// This will register the extension to the extensions menu.
func register_extension(e extensions.Extension) {
	if __extensionsApp == nil {
		extensions_menu_items()
	}

	var extensionName = e.Name()
	__extensionsApp.Children(menu.NewItem(
		extensionName,
		fmt.Sprintf("%s/%s", __extensionsApp.URL.WholeURL, httputils.SimpleSlugify(extensionName)),
		fmt.Sprintf("/%s", httputils.SimpleSlugify(extensionName)),
		0,
	))
}
