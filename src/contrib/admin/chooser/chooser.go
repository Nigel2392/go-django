package chooser

import (
	"context"
	"net/http"
	"reflect"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/core"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/go-signals"
	"github.com/Nigel2392/mux"
	"github.com/elliotchance/orderedmap/v2"
)

type Chooser interface {
	GetTitle(ctx context.Context) string
	GetModel() attrs.Definer
	ListView(adminSite *admin.AdminApplication, app *admin.AppDefinition, model *admin.ModelDefinition) views.View
	CreateView(adminSite *admin.AdminApplication, app *admin.AppDefinition, model *admin.ModelDefinition) views.View
	UpdateView(adminSite *admin.AdminApplication, app *admin.AppDefinition, model *admin.ModelDefinition, instance attrs.Definer) views.View
}

type chooserRegistry struct {
	choosers *orderedmap.OrderedMap[reflect.Type, Chooser]
}

var registry = &chooserRegistry{
	choosers: orderedmap.NewOrderedMap[reflect.Type, Chooser](),
}

var _, _ = core.OnModelsReady.Listen(func(s signals.Signal[any], a any) error {
	if !django.AppInstalled("admin") {
		logger.Error("Admin app is not installed, but chooser forms are being used.")
		return nil
	}

	admin.RegisterModelsRouteHook(func(adminSite *admin.AdminApplication, route mux.Multiplexer) {
		var chooserRoot = route.Any("chooser/", nil, "chooser")

		chooserRoot.Handle(mux.ANY, "list/", admin.NewModelHandler("app_name", "model_name", func(w http.ResponseWriter, r *http.Request, adminSite *admin.AdminApplication, app *admin.AppDefinition, model *admin.ModelDefinition) {
			var modelTyp = reflect.TypeOf(model.Model)
			chooser, ok := registry.choosers.Get(modelTyp)
			if !ok {
				logger.Error("No chooser registered for model type %s", modelTyp)
				except.Fail(
					http.StatusNotFound,
					"Chooser not found for model type %T", model.Model,
				)
				return
			}

			var view = chooser.ListView(adminSite, app, model)
			if view == nil {
				logger.Error("No list view registered for model type %s", modelTyp)
				except.Fail(
					http.StatusNotFound,
					"List view not found for model type %T", model.Model,
				)
				return
			}

			views.Invoke(view, w, r)
		}))

		chooserRoot.Handle(mux.ANY, "create/", admin.NewModelHandler("app_name", "model_name", func(w http.ResponseWriter, r *http.Request, adminSite *admin.AdminApplication, app *admin.AppDefinition, model *admin.ModelDefinition) {
			var instanceObj = model.NewInstance()
			var modelTyp = reflect.TypeOf(instanceObj)
			chooser, ok := registry.choosers.Get(modelTyp)
			if !ok {
				logger.Error("No chooser registered for model type %s", modelTyp)
				except.Fail(
					http.StatusNotFound,
					"Chooser not found for model type %T", instanceObj,
				)
				return
			}

			var view = chooser.CreateView(adminSite, app, model)
			if view == nil {
				logger.Error("No create view registered for model type %s", modelTyp)
				except.Fail(
					http.StatusNotFound,
					"Create view not found for model type %T", instanceObj,
				)
				return
			}

			views.Invoke(view, w, r)
		}))

		chooserRoot.Handle(mux.ANY, "update/<<model_id>>/", admin.NewInstanceHandler("app_name", "model_name", "model_id", func(w http.ResponseWriter, r *http.Request, adminSite *admin.AdminApplication, app *admin.AppDefinition, model *admin.ModelDefinition, instance attrs.Definer) {
			var modelTyp = reflect.TypeOf(instance)
			chooser, ok := registry.choosers.Get(modelTyp)
			if !ok {
				logger.Error("No chooser registered for model type %s", modelTyp)
				except.Fail(
					http.StatusNotFound,
					"Chooser not found for model type %T", instance,
				)
				return
			}

			var view = chooser.UpdateView(adminSite, app, model, instance)
			if view == nil {
				logger.Error("No update view registered for model type %s", modelTyp)
				except.Fail(
					http.StatusNotFound,
					"Update view not found for model type %T", instance,
				)
				return
			}

			views.Invoke(view, w, r)
		}))
	})
	return nil
})

type ChooserDefinition[T attrs.Definer] struct {
	Model T
	Title any // string or func(ctx context.Context) string

	ListPage   *ChooserListPage[T]
	CreatePage *ChooserFormPage[T]
	UpdatePage *ChooserFormPage[T]
}

func (c *ChooserDefinition[T]) GetTitle(ctx context.Context) string {
	switch v := c.Title.(type) {
	case string:
		return v
	case func(ctx context.Context) string:
		return v(ctx)
	}
	assert.Fail("ChooserDefinition.Title must be a string or a function that returns a string")
	return ""
}

func (c *ChooserDefinition[T]) GetModel() attrs.Definer {
	return c.Model
}
