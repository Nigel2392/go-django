package admin

import (
	"reflect"

	"github.com/Nigel2392/django"
	"github.com/Nigel2392/django/contrib/admin/components/menu"
	"github.com/Nigel2392/django/core"
	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/django/forms/media"
	"github.com/Nigel2392/django/views"
	"github.com/Nigel2392/goldcrest"
	"github.com/a-h/templ"
	"github.com/elliotchance/orderedmap/v2"
)

type AdminView interface {
	views.View
	Site() *AdminApplication
}

type AppOptions struct {
	RegisterToAdminMenu bool
	AppLabel            func() string
	AppDescription      func() string
	MenuLabel           func() string
	MenuIcon            func() string
	MediaFn             func() media.Media
	IndexView           func(adminSite *AdminApplication, app *AppDefinition) views.View
}

type AppDefinition struct {
	Name    string
	Options AppOptions
	Models  *orderedmap.OrderedMap[
		string, *ModelDefinition,
	]
	URLs []core.URL
}

func (a *AppDefinition) Register(opts ModelOptions) *ModelDefinition {

	var rTyp = reflect.TypeOf(opts.Model)
	if rTyp.Kind() == reflect.Ptr {
		rTyp = rTyp.Elem()
	}

	assert.False(
		rTyp.Kind() == reflect.Invalid,
		"Model must be a valid type")

	assert.False(
		opts.GetForID == nil,
		"GetForID must be implemented",
	)

	assert.False(
		opts.GetList == nil,
		"GetList must be implemented",
	)

	assert.True(
		rTyp.Kind() == reflect.Struct,
		"Model must be a struct")

	assert.True(
		rTyp.NumField() > 0,
		"Model must have fields")

	var model = &ModelDefinition{
		Name:         opts.GetName(),
		ModelOptions: opts,
		_rModel:      rTyp,
	}

	assert.True(
		model.Name != "",
		"Model must have a name")

	assert.True(
		nameRegex.MatchString(model.Name),
		"Model name must match regex %v",
		nameRegex,
	)

	a.Models.Set(model.Name, model)

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
		var hookFn = RegisterScriptHookFunc(func(adminSite *AdminApplication) media.Media {
			return a.Options.MediaFn()
		})

		goldcrest.Register(RegisterGlobalMedia, 0, hookFn)
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

		var hookFn = func(site *AdminApplication, items menu.Items) {
			var menuItem = &menu.SubmenuItem{
				BaseItem: menu.BaseItem{
					Label: menuLabel,
					Logo:  menuIcon,
				},
				Menu: &menu.Menu{
					Items: make([]menu.MenuItem, 0),
				},
			}

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

			for front := a.Models.Front(); front != nil; front = front.Next() {
				var model = front.Value

				if !model.ModelOptions.RegisterToAdminMenu {
					continue
				}

				menuItem.Menu.Items = append(menuItem.Menu.Items, &menu.Item{
					BaseItem: menu.BaseItem{
						Label: model.Label,
					},
					Link: func() string {
						return django.Reverse("admin:apps:model", a.Name, model.Name)
					},
				})
			}

			items.Append(menuItem)
		}

		goldcrest.Register(RegisterMenuItemHook, 0, hookFn)
	}
}
