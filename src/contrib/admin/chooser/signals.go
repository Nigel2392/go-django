package chooser

import (
	"embed"
	"net/http"
	"reflect"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/core"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/staticfiles"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/go-signals"
	"github.com/Nigel2392/mux"
)

//go:embed assets/**
var choosersFS embed.FS

var _, _ = core.OnDjangoReady.Listen(func(s signals.Signal[any], a any) error {
	for head := choosers.Front(); head != nil; head = head.Next() {
		if err := head.Value.Setup(); err != nil {
			return errors.Wrapf(err, "Error setting up chooser for model type %T", reflect.Zero(head.Key).Interface())
		}
	}
	return nil
})

var _, _ = core.OnModelsReady.Listen(func(s signals.Signal[any], a any) error {
	if !django.AppInstalled("admin") {
		logger.Error("Admin app is not installed, but chooser forms are being used.")
		return nil
	}

	var (
		templateFS = filesystem.Sub(choosersFS, "assets/templates")
		staticFS   = filesystem.Sub(choosersFS, "assets/static")
	)

	tpl.Add(tpl.Config{
		AppName: "chooser",
		FS:      templateFS,
		Bases: []string{
			"chooser/modal/skeleton.tmpl",
			"chooser/modal/controls.tmpl",
			"chooser/modal/modal.tmpl",
		},
	})

	staticfiles.AddFS(staticFS, filesystem.MatchAnd(
		filesystem.MatchPrefix("chooser/"),
		filesystem.MatchOr(
			filesystem.MatchExt(".css"),
			filesystem.MatchExt(".js"),
			filesystem.MatchExt(".png"),
			filesystem.MatchExt(".jpg"),
		),
	))

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
		}), "list")

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
		}), "create")

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
		}), "update")
	})
	return nil
})
