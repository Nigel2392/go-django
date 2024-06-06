package admin

import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/django/core/attrs"
	"github.com/elliotchance/orderedmap/v2"
)

type ModelOptions struct {
	Name    string
	Fields  []string
	Exclude []string
	Model   attrs.Definer
}

func (o *ModelOptions) GetName() string {
	if o.Name == "" {
		var rTyp = reflect.TypeOf(o.Model)
		if rTyp.Kind() == reflect.Ptr {
			return rTyp.Elem().Name()
		}
		return rTyp.Name()
	}
	return o.Name
}

type AppDefinition struct {
	Name   string
	Models *orderedmap.OrderedMap[
		string, *ModelDefinition,
	]
}

func (a *AppDefinition) Register(opts ModelOptions) *ModelDefinition {

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

	var model = &ModelDefinition{
		Name:   opts.GetName(),
		Fields: opts.Fields,
		Model:  rTyp,
	}

	assert.True(
		model.Name != "",
		"Model must have a name")

	a.Models.Set(model.Name, model)

	return model
}

func rt(name string) string {
	return fmt.Sprintf("%s/", name)
}

func (a *AppDefinition) OnReady(adminSite *AdminApplication) {
	var models = a.Models.Keys()
	for _, model := range models {
		var modelDef, ok = a.Models.Get(model)
		assert.True(ok, "Model not found")
		modelDef.OnRegister(adminSite, a)
	}

	//// var routeGroup = mux.NewRoute(mux.ANY, a.Name, nil)
	//var routeGroup = adminSite.Route.Handle(
	//	mux.ANY, rt(a.Name),
	//	AppHandler(adminSite, a),
	//	a.Name,
	//)

	//for _, model := range models {
	//	var modelDef, ok = a.Models.Get(model)
	//	assert.True(ok, "Model not found")
	//
	//	var (
	//		LIST_URL = rt(modelDef.Name)
	//		ADD_URL  = path.Join(modelDef.Name, rt("add"))
	//		EDIT_URL = path.Join(modelDef.Name, rt("edit"))
	//		DEL_URL  = path.Join(modelDef.Name, rt("delete"))
	//
	//		listRoute = routeGroup.Handle(
	//			mux.GET, LIST_URL,
	//			ModelListHandler(adminSite, a, modelDef),
	//			modelDef.Name,
	//		)
	//	)
	//
	//	listRoute.Handle(
	//		mux.GET, ADD_URL,
	//		ModelAddHandler(adminSite, a, modelDef),
	//		"add",
	//	)
	//
	//	listRoute.Handle(
	//		mux.GET, EDIT_URL,
	//		ModelEditHandler(adminSite, a, modelDef),
	//		"edit",
	//	)
	//
	//	listRoute.Handle(
	//		mux.GET, DEL_URL,
	//		ModelDeleteHandler(adminSite, a, modelDef),
	//		"delete",
	//	)
	//}
}

var AppHandler = func(w http.ResponseWriter, r *http.Request, adminSite *AdminApplication, app *AppDefinition) {
	w.Write([]byte(app.Name))
}

var ModelListHandler = func(w http.ResponseWriter, r *http.Request, adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition) {
	w.Write([]byte(model.Name))
	w.Write([]byte("\n"))
	w.Write([]byte("list"))
}

var ModelAddHandler = func(w http.ResponseWriter, r *http.Request, adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition) {
	w.Write([]byte(model.Name))
	w.Write([]byte("\n"))
	w.Write([]byte("add"))
}

var ModelEditHandler = func(w http.ResponseWriter, r *http.Request, adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition, instsance attrs.Definer) {
	w.Write([]byte(model.Name))
	w.Write([]byte("\n"))
	w.Write([]byte("edit"))
}

var ModelDeleteHandler = func(w http.ResponseWriter, r *http.Request, adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition, instsance attrs.Definer) {
	w.Write([]byte(model.Name))
	w.Write([]byte("\n"))
	w.Write([]byte("delete"))
}
