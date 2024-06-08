package admin

import (
	"net/http"

	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/forms"
	"github.com/Nigel2392/django/views"
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

	var addView = &views.FormView[*forms.BaseForm]{
		BaseView: views.BaseView{
			AllowedMethods:  []string{http.MethodGet, http.MethodPost},
			BaseTemplateKey: "admin",
			TemplateName:    "admin/views/add.tmpl",
		},
		GetFormFn: func(req *http.Request) *forms.BaseForm {
			var form = forms.NewBaseForm()
			var instance = model.NewInstance()
			for _, field := range model.ModelFields(instance) {
				form.AddField(
					field.Name(), field.FormField(),
				)
			}
			return form
		},
	}

	views.Invoke(addView, w, r)
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
