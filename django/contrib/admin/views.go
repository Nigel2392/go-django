package admin

import (
	"fmt"
	"net/http"

	"github.com/Nigel2392/django"
	"github.com/Nigel2392/django/core"
	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/core/ctx"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/django/forms/modelforms"
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
			model.GetLabel(field, field), model.FormatColumn(field),
		)
	}

	var view = &list.View[attrs.Definer]{
		ListColumns: columns,
		BaseView: views.BaseView{
			AllowedMethods:  []string{http.MethodGet, http.MethodPost},
			BaseTemplateKey: "admin",
			TemplateName:    "admin/views/models/list.tmpl",
			GetContextFn: func(req *http.Request) (ctx.Context, error) {
				var context = core.Context(req)
				context.Set("app", app)
				context.Set("model", model)
				return context, nil
			},
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

func getFormForInstance(instance attrs.Definer, app *AppDefinition, model *ModelDefinition, r *http.Request) modelforms.ModelForm[attrs.Definer] {
	var form modelforms.ModelForm[attrs.Definer]
	if f, ok := instance.(FormDefiner); ok {
		form = f.AdminForm(r, app, model)
	} else {
		form = modelforms.NewBaseModelForm[attrs.Definer](instance)
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
	return form
}

func newInstanceView(instance attrs.Definer, app *AppDefinition, model *ModelDefinition, r *http.Request) *views.FormView[modelforms.ModelForm[attrs.Definer]] {
	return &views.FormView[modelforms.ModelForm[attrs.Definer]]{
		BaseView: views.BaseView{
			AllowedMethods:  []string{http.MethodGet, http.MethodPost},
			BaseTemplateKey: "admin",
			TemplateName:    "admin/views/models/edit.tmpl",
		},
		GetFormFn: func(req *http.Request) modelforms.ModelForm[attrs.Definer] {
			var form = getFormForInstance(instance, app, model, r)
			form.SetInstance(instance)
			return form
		},
		GetInitialFn: func(req *http.Request) map[string]interface{} {
			var initial = make(map[string]interface{})
			if instance == nil {
				fmt.Println(initial, "is nil")
				return initial
			}
			for _, def := range model.ModelFields(instance) {
				var v = def.GetValue()
				var n = def.Name()
				if fields.IsZero(v) {
					initial[n] = def.GetDefault()
				} else {
					initial[n] = v
				}
			}
			fmt.Println(initial)
			return initial
		},
	}
}

var ModelAddHandler = func(w http.ResponseWriter, r *http.Request, adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition) {
	var instance = model.NewInstance()
	var addView = newInstanceView(instance, app, model, r)
	views.Invoke(addView, w, r)
}

var ModelEditHandler = func(w http.ResponseWriter, r *http.Request, adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition, instance attrs.Definer) {
	var editView = newInstanceView(instance, app, model, r)
	views.Invoke(editView, w, r)
}

var ModelDeleteHandler = func(w http.ResponseWriter, r *http.Request, adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition, instance attrs.Definer) {
	w.Write([]byte(model.Name))
	w.Write([]byte("\n"))
	w.Write([]byte("delete"))
}
