package chooser

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"reflect"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/core"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/go-signals"
	"github.com/Nigel2392/mux"
	"github.com/elliotchance/orderedmap/v2"
)

type Chooser interface {
	Setup() error
	GetTitle(ctx context.Context) string
	GetModel() attrs.Definer
	ListView(adminSite *admin.AdminApplication, app *admin.AppDefinition, model *admin.ModelDefinition) views.View
	CreateView(adminSite *admin.AdminApplication, app *admin.AppDefinition, model *admin.ModelDefinition) views.View
	UpdateView(adminSite *admin.AdminApplication, app *admin.AppDefinition, model *admin.ModelDefinition, instance attrs.Definer) views.View
}

var choosers = orderedmap.NewOrderedMap[reflect.Type, Chooser]()

var _, _ = core.OnModelsReady.Listen(func(s signals.Signal[any], a any) error {
	if !django.AppInstalled("admin") {
		logger.Error("Admin app is not installed, but chooser forms are being used.")
		return nil
	}

	for head := choosers.Front(); head != nil; head = head.Next() {
		if err := head.Value.Setup(); err != nil {
			return errors.Wrapf(err, "Error setting up chooser for model type %T", reflect.Zero(head.Key).Interface())
		}
	}

	admin.RegisterModelsRouteHook(func(adminSite *admin.AdminApplication, route mux.Multiplexer) {
		var chooserRoot = route.Any("chooser/", nil, "chooser")

		chooserRoot.Handle(mux.ANY, "list/", admin.NewModelHandler("app_name", "model_name", func(w http.ResponseWriter, r *http.Request, adminSite *admin.AdminApplication, app *admin.AppDefinition, model *admin.ModelDefinition) {
			var modelTyp = reflect.TypeOf(model.Model)
			chooser, ok := choosers.Get(modelTyp)
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
			chooser, ok := choosers.Get(modelTyp)
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
			chooser, ok := choosers.Get(modelTyp)
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

func (c *ChooserDefinition[T]) Setup() error {
	if c.Title == nil {
		return errors.ValueError.Wrap("ChooserDefinition.Title cannot be nil")
	}

	if reflect.ValueOf(c.Model).IsNil() {
		return errors.TypeMismatch.Wrap("ChooserDefinition.Model cannot be nil")
	}

	if c.ListPage != nil {
		c.ListPage._Definition = c
	}

	if c.CreatePage != nil {
		c.CreatePage._Definition = c
	}

	if c.UpdatePage != nil {
		c.UpdatePage._Definition = c
	}

	return nil
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

func (c *ChooserDefinition[T]) ListView(adminSite *admin.AdminApplication, app *admin.AppDefinition, model *admin.ModelDefinition) views.View {
	if c.ListPage != nil {
		c.ListPage._Definition = c
	}
	return c.ListPage
}

func (c *ChooserDefinition[T]) CreateView(adminSite *admin.AdminApplication, app *admin.AppDefinition, model *admin.ModelDefinition) views.View {
	if c.CreatePage != nil {
		c.CreatePage._Definition = c
	}
	return c.CreatePage
}

func (c *ChooserDefinition[T]) UpdateView(adminSite *admin.AdminApplication, app *admin.AppDefinition, model *admin.ModelDefinition, instance attrs.Definer) views.View {
	if c.UpdatePage != nil {
		c.UpdatePage._Definition = c
	}
	return c.UpdatePage
}

func (c *ChooserDefinition[T]) GetContext(req *http.Request, page, bound views.View) ctx.Context {
	var ctx = ctx.RequestContext(req)
	ctx.Set("chooser", c)
	ctx.Set("chooser_page", page)
	ctx.Set("chooser_view", bound)
	return ctx
}

type JSONHtmlResponse struct {
	HTML string `json:"html"`
}

func (c *ChooserDefinition[T]) Render(w http.ResponseWriter, req *http.Request, context ctx.Context, base, template string) error {
	var buf = new(bytes.Buffer)
	if err := tpl.FRender(buf, context, base, template); err != nil {
		return err
	}

	var response = JSONHtmlResponse{
		HTML: buf.String(),
	}

	return json.NewEncoder(w).Encode(response)
}
