package admin

import (
	"fmt"
	"net/http"

	"github.com/Nigel2392/django"
	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/forms"
	"github.com/Nigel2392/django/views"
	"github.com/Nigel2392/django/views/list"
)

var AppHandler = func(w http.ResponseWriter, r *http.Request, adminSite *AdminApplication, app *AppDefinition) {
	w.Write([]byte(app.Name))
}

var ModelListHandler = func(w http.ResponseWriter, r *http.Request, adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition) {

	// var instances, err = model.GetList(10, 0)

	var columns = make([]list.ListColumn[attrs.Definer], len(model.Fields))
	for i, field := range model.Fields {
		columns[i] = list.Column[attrs.Definer](
			model.GetLabel(field, field), field,
		)
	}

	fmt.Println(columns, model.Fields)

	var view = &list.View[attrs.Definer]{
		ListColumns: columns,
		BaseView: views.BaseView{
			AllowedMethods:  []string{http.MethodGet, http.MethodPost},
			BaseTemplateKey: "admin",
			TemplateName:    "admin/views/models/list.tmpl",
		},
		GetListFn: func(amount, offset uint, include []string) ([]attrs.Definer, error) {
			return model.GetList(amount, offset, include)
		},
		TitleFieldColumn: func(lc list.ListColumn[attrs.Definer]) list.ListColumn[attrs.Definer] {
			return list.TitleFieldColumn(lc, func(defs attrs.Definitions, instance attrs.Definer) string {
				var primaryField = defs.Primary()
				if primaryField == nil {
					return ""
				}
				return django.Reverse("admin:apps:model:edit", app.Name, model.Name, primaryField.GetValue())
			})
		},
	}

	views.Invoke(view, w, r)
}

var ModelAddHandler = func(w http.ResponseWriter, r *http.Request, adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition) {

	var form forms.Form
	var instance = model.NewInstance()
	if f, ok := instance.(FormDefiner); ok {
		form = f.AdminForm(r, app, model)
	} else {
		form = forms.NewBaseForm()
		for _, field := range model.ModelFields(instance) {
			var (
				name      = field.Name()
				formfield = field.FormField()
			)

			var label = model.GetLabel(name, field.Label())
			formfield.SetLabel(label)

			form.AddField(
				name, formfield,
			)
		}
	}

	var addView = &views.FormView[forms.Form]{
		BaseView: views.BaseView{
			AllowedMethods:  []string{http.MethodGet, http.MethodPost},
			BaseTemplateKey: "admin",
			TemplateName:    "admin/views/models/add.tmpl",
		},
		GetFormFn: func(req *http.Request) forms.Form {
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
