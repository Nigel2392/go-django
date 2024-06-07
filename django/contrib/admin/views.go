package admin

import (
	"net/http"

	"github.com/Nigel2392/django/core/attrs"
)

var AppHandler = func(w http.ResponseWriter, r *http.Request, adminSite *AdminApplication, app *AppDefinition) {
	w.Write([]byte(app.Name))
}

var ModelListHandler = func(w http.ResponseWriter, r *http.Request, adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition) {

	// var instances, err = model.GetList(10, 0)

	w.Write([]byte(model.Name))
	w.Write([]byte("\n"))
	w.Write([]byte("list"))
}

var ModelAddHandler = func(w http.ResponseWriter, r *http.Request, adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition) {
	w.Write([]byte(model.Name))
	w.Write([]byte("\n"))
	w.Write([]byte("add"))
}

var ModelEditHandler = func(w http.ResponseWriter, r *http.Request, adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition, instance attrs.Definer) {
	w.Write([]byte(model.Name))
	w.Write([]byte("\n"))
	w.Write([]byte("edit"))
}

var ModelDeleteHandler = func(w http.ResponseWriter, r *http.Request, adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition, instance attrs.Definer) {
	w.Write([]byte(model.Name))
	w.Write([]byte("\n"))
	w.Write([]byte("delete"))
}
