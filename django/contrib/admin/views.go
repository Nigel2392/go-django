package admin

import (
	"fmt"
	"net/http"

	"github.com/Nigel2392/django"
	"github.com/Nigel2392/django/contrib/auth"
	"github.com/Nigel2392/django/core"
	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/core/ctx"
	"github.com/Nigel2392/django/core/tpl"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/django/forms/modelforms"
	"github.com/Nigel2392/django/permissions"
	"github.com/Nigel2392/django/views"
	"github.com/Nigel2392/django/views/list"
)

var AppHandler = func(w http.ResponseWriter, r *http.Request, adminSite *AdminApplication, app *AppDefinition) {
	if app.Options.IndexView != nil {
		var err = views.Invoke(app.Options.IndexView(adminSite, app), w, r)
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
	context.SetPage(PageOptions{
		TitleFn:    app.Label,
		SubtitleFn: app.Description,
	})
	context.Set("app", app)
	context.Set("models", modelNames)
	var err = tpl.FRender(w, context, BASE_KEY, "admin/views/apps/index.tmpl")
	assert.Err(err)
}

var ModelListHandler = func(w http.ResponseWriter, r *http.Request, adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition) {

	if !permissions.Object(r, "admin:view_list", model.NewInstance()) {
		auth.Fail(
			http.StatusForbidden,
			"Permission denied",
		)
		return
	}

	if model.ListView.GetHandler != nil {
		var err = views.Invoke(model.ListView.GetHandler(adminSite, app, model), w, r)
		assert.Err(err)
		return
	}

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
			BaseTemplateKey: BASE_KEY,
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

	if !permissions.Object(r, "admin:add", model.NewInstance()) {
		auth.Fail(
			http.StatusForbidden,
			"Permission denied",
		)
		return
	}

	var instance = model.NewInstance()
	if model.AddView.GetHandler != nil {
		var err = views.Invoke(model.AddView.GetHandler(adminSite, app, model, instance), w, r)
		assert.Err(err)
		return
	}
	var addView = newInstanceView("add", instance, model.AddView, app, model, r)
	views.Invoke(addView, w, r)
	// if err := views.Invoke(addView, w, r); err != nil {
	// panic(err)
	// }
}

var ModelEditHandler = func(w http.ResponseWriter, r *http.Request, adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition, instance attrs.Definer) {
	if !permissions.Object(r, "admin:edit", instance) {
		auth.Fail(
			http.StatusForbidden,
			"Permission denied",
		)
		return
	}

	if model.EditView.GetHandler != nil {
		var err = views.Invoke(model.EditView.GetHandler(adminSite, app, model, instance), w, r)
		assert.Err(err)
		return
	}
	var editView = newInstanceView("edit", instance, model.EditView, app, model, r)
	views.Invoke(editView, w, r)
	// if err := views.Invoke(editView, w, r); err != nil {
	// panic(err)
	// }
}

var ModelDeleteHandler = func(w http.ResponseWriter, r *http.Request, adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition, instance attrs.Definer) {
	if !permissions.Object(r, "admin:delete", instance) {
		auth.Fail(
			http.StatusForbidden,
			"Permission denied",
		)
		return
	}

	if model.DeleteView.GetHandler != nil {
		var err = views.Invoke(model.DeleteView.GetHandler(adminSite, app, model, instance), w, r)
		assert.Err(err)
		return
	}
	w.Write([]byte(model.Name))
	w.Write([]byte("\n"))
	w.Write([]byte("delete"))
}

func GetAdminForm(instance attrs.Definer, opts FormViewOptions, app *AppDefinition, model *ModelDefinition, r *http.Request) modelforms.ModelForm[attrs.Definer] {
	var form modelforms.ModelForm[attrs.Definer]
	if f, ok := instance.(FormDefiner); ok {
		form = f.AdminForm(r, app, model)
	} else {
		var modelForm = modelforms.NewBaseModelForm[attrs.Definer](instance)
		for _, field := range model.ModelFields(opts.ViewOptions, instance) {
			var (
				name      = field.Name()
				formfield = field.FormField()
			)

			var label = model.GetLabel(opts.ViewOptions, name, field.Label())
			formfield.SetLabel(label)

			modelForm.AddField(
				name, formfield,
			)
		}
		modelForm.SaveInstance = opts.SaveInstance
		form = modelForm
	}
	return form
}

func newInstanceView(tpl string, instance attrs.Definer, opts FormViewOptions, app *AppDefinition, model *ModelDefinition, r *http.Request) *views.FormView[*AdminModelForm[modelforms.ModelForm[attrs.Definer]]] {
	return &views.FormView[*AdminModelForm[modelforms.ModelForm[attrs.Definer]]]{
		BaseView: views.BaseView{
			AllowedMethods:  []string{http.MethodGet, http.MethodPost},
			BaseTemplateKey: BASE_KEY,
			TemplateName:    fmt.Sprintf("admin/views/models/%s.tmpl", tpl),
			GetContextFn: func(req *http.Request) (ctx.Context, error) {
				var context = core.Context(req)
				context.Set("app", app)
				context.Set("model", model)
				return context, nil
			},
		},
		GetFormFn: func(req *http.Request) *AdminModelForm[modelforms.ModelForm[attrs.Definer]] {
			var form modelforms.ModelForm[attrs.Definer]
			if opts.GetForm != nil {
				form = opts.GetForm(req, instance, opts.ViewOptions.Fields)
			} else {
				form = GetAdminForm(instance, opts, app, model, r)
				form.SetFields(attrs.FieldNames(instance, opts.Exclude)...)
			}

			if opts.FormInit != nil {
				opts.FormInit(instance, form)
			}

			form.SetInstance(instance)

			var adminForm = &AdminModelForm[modelforms.ModelForm[attrs.Definer]]{
				AdminForm: &AdminForm[modelforms.ModelForm[attrs.Definer]]{
					Form:   form,
					Panels: opts.Panels,
				},
			}

			var fields = make(map[string]struct{})
			for _, panel := range adminForm.Panels {
				for _, field := range panel.Fields() {
					fields[field] = struct{}{}
				}
			}

			for _, field := range adminForm.Fields() {
				if _, ok := fields[field.Name()]; !ok {
					adminForm.DeleteField(field.Name())
				}
			}

			return adminForm
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
		SuccessFn: func(w http.ResponseWriter, req *http.Request, form *AdminModelForm[modelforms.ModelForm[attrs.Definer]]) {
			var instance = form.Instance()
			assert.False(instance == nil, "instance is nil after form submission")
			var listViewURL = django.Reverse("admin:apps:model", app.Name, model.Name)
			http.Redirect(w, r, listViewURL, http.StatusSeeOther)
		},
	}
}
