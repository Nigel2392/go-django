package admin

import (
	"fmt"
	"net/http"

	"github.com/Nigel2392/django"
	"github.com/Nigel2392/django/core"
	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/core/ctx"
	"github.com/Nigel2392/django/core/tpl"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/django/forms/modelforms"
	"github.com/Nigel2392/django/views"
	"github.com/Nigel2392/django/views/list"
)

var AppHandler = func(w http.ResponseWriter, r *http.Request, adminSite *AdminApplication, app *AppDefinition) {
	if app.Options.IndexView != nil {
		var err = views.Invoke(app.Options.IndexView, w, r)
		assert.Err(err)
		return
	}

	var models = app.Models
	var modelNames = make([]string, models.Len())
	var i = 0
	for front := models.Front(); front != nil; front = front.Next() {
		modelNames[i] = front.Value.Label()
		i++
	}

	var context = NewContext(
		r, adminSite, nil,
	)
	context.Set("app", app)
	context.Set("models", modelNames)
	var err = tpl.FRender(w, context, "admin/views/apps/index.tmpl")
	assert.Err(err)
}

var ModelListHandler = func(w http.ResponseWriter, r *http.Request, adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition) {

	// var instances, err = model.GetList(10, 0)

	var columns = make([]list.ListColumn[attrs.Definer], len(model.ListView.Fields))
	for i, field := range model.ListView.Fields {
		columns[i] = model.GetColumn(model.ListView, field)
	}

	var amount = model.ListView.PerPage
	if amount == 0 {
		amount = 25
	}

	var view = &list.View[attrs.Definer]{
		ListColumns:   columns,
		DefaultAmount: amount,
		BaseView: views.BaseView{
			AllowedMethods:  []string{http.MethodGet, http.MethodPost},
			BaseTemplateKey: "admin",
			TemplateName:    "admin/views/models/list.tmpl",
			GetContextFn: func(req *http.Request) (ctx.Context, error) {
				var context = NewContext(req, adminSite, nil)
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

var ModelAddHandler = func(w http.ResponseWriter, r *http.Request, adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition) {
	var instance = model.NewInstance()
	var addView = newInstanceView("add", instance, model.AddView, app, model, r)
	views.Invoke(addView, w, r)
	// if err := views.Invoke(addView, w, r); err != nil {
	// panic(err)
	// }
}

var ModelEditHandler = func(w http.ResponseWriter, r *http.Request, adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition, instance attrs.Definer) {
	var editView = newInstanceView("edit", instance, model.EditView, app, model, r)
	views.Invoke(editView, w, r)
	// if err := views.Invoke(editView, w, r); err != nil {
	// panic(err)
	// }
}

var ModelDeleteHandler = func(w http.ResponseWriter, r *http.Request, adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition, instance attrs.Definer) {
	w.Write([]byte(model.Name))
	w.Write([]byte("\n"))
	w.Write([]byte("delete"))
}

func getFormForInstance(instance attrs.Definer, opts ViewOptions, app *AppDefinition, model *ModelDefinition, r *http.Request) modelforms.ModelForm[attrs.Definer] {
	var form modelforms.ModelForm[attrs.Definer]
	if f, ok := instance.(FormDefiner); ok {
		form = f.AdminForm(r, app, model)
	} else {
		form = modelforms.NewBaseModelForm[attrs.Definer](instance)
		for _, field := range model.ModelFields(opts, instance) {
			var (
				name      = field.Name()
				formfield = field.FormField()
			)

			var label = model.GetLabel(opts, name, field.Label())
			formfield.SetLabel(label)

			form.AddField(
				name, formfield,
			)
		}
	}
	return form
}

func newInstanceView(tpl string, instance attrs.Definer, opts FormViewOptions, app *AppDefinition, model *ModelDefinition, r *http.Request) *views.FormView[modelforms.ModelForm[attrs.Definer]] {
	return &views.FormView[modelforms.ModelForm[attrs.Definer]]{
		BaseView: views.BaseView{
			AllowedMethods:  []string{http.MethodGet, http.MethodPost},
			BaseTemplateKey: "admin",
			TemplateName:    fmt.Sprintf("admin/views/models/%s.tmpl", tpl),
			GetContextFn: func(req *http.Request) (ctx.Context, error) {
				var context = core.Context(req)
				context.Set("app", app)
				context.Set("model", model)
				return context, nil
			},
		},
		GetFormFn: func(req *http.Request) modelforms.ModelForm[attrs.Definer] {
			var form modelforms.ModelForm[attrs.Definer]
			if opts.GetForm != nil {
				form = opts.GetForm(req, instance, opts.ViewOptions.Fields)
			} else {
				form = getFormForInstance(instance, opts.ViewOptions, app, model, r)
				form.SetFields(attrs.FieldNames(instance, opts.Exclude)...)
			}

			if opts.FormInit != nil {
				opts.FormInit(instance, form)
			}

			form.SetInstance(instance)
			return form
		},
		GetInitialFn: func(req *http.Request) map[string]interface{} {
			var initial = make(map[string]interface{})
			if instance == nil {
				return initial
			}
			for _, def := range model.ModelFields(opts.ViewOptions, instance) {
				var v = def.GetValue()
				var n = def.Name()
				if fields.IsZero(v) {
					initial[n] = def.GetDefault()
				} else {
					initial[n] = v
				}
			}
			return initial
		},
		SuccessFn: func(w http.ResponseWriter, req *http.Request, form modelforms.ModelForm[attrs.Definer]) {
			var instance = form.Instance()
			assert.False(instance == nil, "instance is nil after form submission")
			var listViewURL = django.Reverse("admin:apps:model", app.Name, model.Name)
			http.Redirect(w, r, listViewURL, http.StatusSeeOther)
		},
	}
}
