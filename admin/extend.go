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

// The internal menu.
//
// Will only be loaded once.
func (a *AdminSite) internal_menu_items() *menu.Item {
	if a.internalApp != nil {
		return a.internalApp
	}

	a.internalApp = menu.NewItem(
		a.InternalAppName,
		fmt.Sprintf("%s/%s", a.URL, a.InternalAppName),
		fmt.Sprintf("/%s", a.InternalAppName),
		0,
	)

	return a.internalApp
}

// The extensions menu.
//
// Will only be loaded once.
func (a *AdminSite) extensions_menu_items() *menu.Item {
	if a.extensionsApp != nil {
		return a.extensionsApp
	}

	a.extensionsApp = menu.NewItem(
		a.ExtensionsAppName,
		httputils.NicePath(false, a.URL, a.ExtensionURL),
		httputils.NicePath(false, a.ExtensionURL),
		0,
	)

	return a.extensionsApp
}

// Register an internal model.
//
// This will register the model to the internal menu.
func (a *AdminSite) register_internal_model(m *internalModel) {
	if a.internalApp == nil {
		a.internal_menu_items()
	}

	if m.perms == nil {
		m.perms = auth.PermissionMap{
			"view": auth.NewPermission("view", m.model),
			"list": auth.NewPermission("list", m.model),
		}
	}

	var modelName = namer.GetModelName(m.model)
	a.internalApp.Children(menu.NewItem(
		httputils.TitleCaser.String(modelName),
		fmt.Sprintf("%s/%s", a.internalApp.URL.WholeURL, httputils.SimpleSlugify(modelName)),
		fmt.Sprintf("/%s", httputils.SimpleSlugify(modelName)),
		0,
		m,
	))
}

// Register an extension.
//
// This will register the extension to the extensions menu.
func (a *AdminSite) register_extension(e extensions.Extension) {
	if a.extensionsApp == nil {
		a.extensions_menu_items()
	}

	var extensionName = e.Name()
	a.extensionsApp.Children(menu.NewItem(
		extensionName,
		fmt.Sprintf("%s/%s", a.extensionsApp.URL.WholeURL, httputils.SimpleSlugify(extensionName)),
		fmt.Sprintf("/%s", httputils.SimpleSlugify(extensionName)),
		0,
	))
}
