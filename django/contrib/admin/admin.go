package admin

import (
	"net/http"

	"github.com/Nigel2392/django"
	"github.com/Nigel2392/django/apps"
	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/mux"
	"github.com/elliotchance/orderedmap/v2"
)

var AdminSite *AdminApplication = &AdminApplication{
	AppConfig: apps.NewAppConfig("admin"),
	Apps: orderedmap.NewOrderedMap[
		string, *AppDefinition,
	](),
	Ordering: make([]string, 0),
	Route: mux.NewRoute(
		mux.ANY, "admin/", mux.NewHandler(func(w http.ResponseWriter, req *http.Request) {
			w.Write([]byte("Admin Interface"))
		}),
	),
}

var (
	RegisterApp = AdminSite.RegisterApp
	IsReady     = AdminSite.IsReady
)

func NewAppConfig() django.AppConfig {
	AdminSite.Init = func(settings django.Settings) error {
		settings.App().Mux.AddRoute(AdminSite.Route)
		return nil
	}

	AdminSite.Ready = func() error {

		var routeApps = AdminSite.Route.Handle(
			mux.ANY, "apps/<<app_name>>/",
			newAppHandler(AppHandler),
		)

		var routeModelsList = routeApps.Handle(
			mux.ANY, "model/<<model_name>>/",
			newModelHandler(ModelListHandler),
		)

		routeModelsList.Handle(
			mux.ANY, "add/",
			newModelHandler(ModelAddHandler),
		)

		routeModelsList.Handle(
			mux.ANY, "edit/<<model_id>>/",
			newInstanceHandler(ModelEditHandler),
		)

		routeModelsList.Handle(
			mux.ANY, "delete/<<model_id>>/",
			newInstanceHandler(ModelDeleteHandler),
		)

		AdminSite.ready.Store(true)

		return nil
	}

	return AdminSite
}

func newHandler(handler func(w http.ResponseWriter, r *http.Request)) mux.Handler {
	return mux.NewHandler(func(w http.ResponseWriter, req *http.Request) {
		handler(w, req)
	})
}

func newInstanceHandler(handler func(w http.ResponseWriter, req *http.Request, adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition, instance attrs.Definer)) mux.Handler {
	return newHandler(func(w http.ResponseWriter, req *http.Request) {
		var (
			vars      = mux.Vars(req)
			appName   = vars.Get("app_name")
			modelName = vars.Get("model_name")
			modelID   = vars.Get("model_id")
		)

		if modelName == "" || appName == "" || modelID == "" {
			http.Error(w, "App, Model name and Model ID is required", http.StatusBadRequest)
			return
		}

		var app, ok = AdminSite.Apps.Get(appName)
		if !ok {
			http.Error(w, "Model not found", http.StatusNotFound)
			return
		}

		model, ok := app.Models.Get(modelName)
		if !ok {
			http.Error(w, "Model not found", http.StatusNotFound)
			return
		}

		var instance, err = model.GetInstance(modelID)
		if err != nil {
			http.Error(w, "Error retrieving model, does it exist?", http.StatusInternalServerError)
			return
		}

		handler(w, req, AdminSite, app, model, instance)
	})
}

func newModelHandler(handler func(w http.ResponseWriter, r *http.Request, adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition)) mux.Handler {
	return newHandler(func(w http.ResponseWriter, req *http.Request) {
		var (
			vars      = mux.Vars(req)
			appName   = vars.Get("app_name")
			modelName = vars.Get("model_name")
		)

		if modelName == "" || appName == "" {
			http.Error(w, "App and Model name is required", http.StatusBadRequest)
			return
		}

		var app, ok = AdminSite.Apps.Get(appName)
		if !ok {
			http.Error(w, "Model not found", http.StatusNotFound)
			return
		}

		model, ok := app.Models.Get(modelName)
		if !ok {
			http.Error(w, "Model not found", http.StatusNotFound)
			return
		}

		handler(w, req, AdminSite, app, model)
	})
}

func newAppHandler(handler func(w http.ResponseWriter, r *http.Request, adminSite *AdminApplication, app *AppDefinition)) mux.Handler {
	return newHandler(func(w http.ResponseWriter, req *http.Request) {
		var vars = mux.Vars(req)
		var appName = vars.Get("app_name")

		if appName == "" {
			http.Error(w, "App name is required", http.StatusBadRequest)
			return
		}

		var app, ok = AdminSite.Apps.Get(appName)
		if !ok {
			http.Error(w, "App not found", http.StatusNotFound)
			return
		}

		handler(w, req, AdminSite, app)
	})
}
