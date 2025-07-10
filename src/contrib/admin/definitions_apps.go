package admin

import (
	"fmt"
	"net/http"
	"reflect"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/admin/components"
	"github.com/Nigel2392/go-django/src/contrib/admin/components/menu"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/go-django/src/permissions"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/goldcrest"
	"github.com/a-h/templ"
	"github.com/elliotchance/orderedmap/v2"
)

type AppOptions struct {
	// RegisterToAdminMenu allows for registering this app to the admin menu by default.
	RegisterToAdminMenu bool

	// EnableIndexView allows for enabling the index view for this app.
	//
	// If this is disabled, only a main sub-menu item will be created, but not for the index view.
	EnableIndexView bool

	// Applabel must return a human readable label for the app.
	AppLabel func() string

	// AppDescription must return a human readable description of this app.
	AppDescription func() string

	// MenuLabel must return a human readable label for the menu, this is how the app's name will appear in the admin's navigation.
	MenuLabel func() string

	// MenuOrder is the order in which the app will appear in the admin's navigation.
	MenuOrder int

	// MenuIcon must return a string representing the icon to use for the menu.
	//
	// This should be a HTML element, I.E. "<svg>...</svg>".
	MenuIcon func() string

	// MediaFn must return a media.Media object that will be used for this app.
	//
	// It will always be included in the admin's media.
	MediaFn func() media.Media

	// A custom index view for the app.
	//
	// This will override the default index view for the app.
	IndexView func(adminSite *AdminApplication, app *AppDefinition) views.View
}

type AppDefinition struct {
	Name    string
	Options AppOptions
	Models  *orderedmap.OrderedMap[
		string, *ModelDefinition,
	]
	Routing func(django.Mux)
}

func (a *AppDefinition) Register(opts ModelOptions) *ModelDefinition {

	if opts.Model == nil {
		logger.Warnf(
			"Model is nil, cannot register model %q",
			opts.Name,
		)
		return nil
	}

	var rTyp = reflect.TypeOf(opts.Model)
	if rTyp.Kind() == reflect.Ptr {
		rTyp = rTyp.Elem()
	}

	assert.False(
		rTyp.Kind() == reflect.Invalid,
		"Model must be a valid type")

	assert.True(
		rTyp.Kind() == reflect.Struct,
		"Model must be a struct")

	assert.True(
		rTyp.NumField() > 0,
		"Model must have fields")

	var cType = contenttypes.DefinitionForObject(
		opts.Model,
	)
	assert.True(
		cType != nil,
		"Model must have a registered content type definition",
	)

	var model = &ModelDefinition{
		ModelOptions: opts,
		_app:         a,
		_rModel:      rTyp,
		_cType:       cType,
	}

	assert.True(
		model.GetName() != "",
		"Model must have a name")

	assert.True(
		nameRegex.MatchString(model.GetName()),
		"Model name must match regex %v",
		nameRegex,
	)

	a.Models.Set(model.GetName(), model)

	return model
}

func (a *AppDefinition) Label() string {
	if a.Options.AppLabel != nil {
		return a.Options.AppLabel()
	}
	return a.Name
}

func (a *AppDefinition) Description() string {
	if a.Options.AppDescription != nil {
		return a.Options.AppDescription()
	}
	return ""
}

func (a *AppDefinition) OnReady(adminSite *AdminApplication) {
	var models = a.Models.Keys()
	for _, model := range models {
		var modelDef, ok = a.Models.Get(model)
		assert.True(ok, "Model not found")
		modelDef.OnRegister(adminSite, a)
	}

	if a.Options.MediaFn != nil {
		var hookFn = RegisterMediaHookFunc(func(adminSite *AdminApplication) media.Media {
			return a.Options.MediaFn()
		})

		goldcrest.Register(RegisterGlobalMediaHook, 0, hookFn)
	}

	if a.Options.RegisterToAdminMenu {
		var menuLabel = a.Options.MenuLabel
		if menuLabel == nil {
			menuLabel = func() string {
				return a.Name
			}
		}

		var menuIcon templ.Component
		if a.Options.MenuIcon != nil {
			menuIcon = templ.Raw(
				a.Options.MenuIcon(),
			)
		}

		var hookFn = func(r *http.Request, site *AdminApplication, items components.Items[menu.MenuItem]) {
			var menuItem = &menu.SubmenuItem{
				BaseItem: menu.BaseItem{
					ItemName: a.Name,
					Label:    menuLabel,
					Logo:     menuIcon,
					Ordering: a.Options.MenuOrder,
					Hidden: !permissions.HasPermission(
						r, fmt.Sprintf("admin:view_app:%s", a.Name),
					),
				},
				Menu: &menu.Menu{
					Items: make([]menu.MenuItem, 0),
				},
			}

			if a.Options.EnableIndexView {
				var menuLabel func() string = a.Options.MenuLabel
				if menuLabel == nil {
					menuLabel = func() string {
						return a.Name
					}
				}
				menuItem.Menu.Items = append(menuItem.Menu.Items, &menu.Item{
					BaseItem: menu.BaseItem{
						Label: menuLabel,
					},
					Link: func() string {
						return django.Reverse("admin:apps", a.Name)
					},
				})
			}

			//	var hooks = goldcrest.Get[RegisterAppMenuItemHookFunc](
			//		fmt.Sprintf("%s:%s", RegisterMenuItemHook, a.Name),
			//	)
			//
			//	for _, hook := range hooks {
			//		if hook == nil {
			//			continue
			//		}
			//
			//		var items = hook(site, a)
			//		if len(items) == 0 {
			//			continue
			//		}
			//
			//		menuItem.Menu.Items = append(menuItem.Menu.Items, items...)
			//	}

			for front := a.Models.Front(); front != nil; front = front.Next() {
				var model = front.Value

				if !model.ModelOptions.RegisterToAdminMenu {
					continue
				}

				var menuIcon templ.Component
				if model.ModelOptions.MenuIcon != nil {
					menuIcon = templ.Raw(
						model.ModelOptions.MenuIcon(),
					)
				}

				menuItem.Menu.Items = append(menuItem.Menu.Items, &menu.Item{
					BaseItem: menu.BaseItem{
						Label:    model.getMenuLabel,
						Ordering: model.MenuOrder,
						Logo:     menuIcon,
						Hidden: !permissions.HasObjectPermission(
							r, model.NewInstance(), "admin:view_list",
						),
					},
					Link: func() string {
						return django.Reverse("admin:apps:model", a.Name, model.GetName())
					},
				})
			}

			items.Append(menuItem)
		}

		goldcrest.Register(RegisterMenuItemHook, 0, RegisterMenuItemHookFunc(hookFn))
	}
}
