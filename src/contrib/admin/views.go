package admin

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	django "github.com/Nigel2392/go-django/src"
	autherrors "github.com/Nigel2392/go-django/src/contrib/auth/auth_errors"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
	"github.com/Nigel2392/go-django/src/models"
	"github.com/Nigel2392/go-django/src/permissions"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/go-django/src/views/list"
	"github.com/Nigel2392/goldcrest"
)

var AppHandler = func(w http.ResponseWriter, r *http.Request, adminSite *AdminApplication, app *AppDefinition) {
	if app.Options.IndexView != nil {
		var err = views.Invoke(app.Options.IndexView(adminSite, app), w, r)
		except.AssertNil(err, 500, err)
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
	except.AssertNil(err, 500, err)
}

var ModelListHandler = func(w http.ResponseWriter, r *http.Request, adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition) {

	if !permissions.HasObjectPermission(r, model.NewInstance(), "admin:view_list") {
		autherrors.Fail(
			http.StatusForbidden,
			"Permission denied",
		)
		return
	}

	if model.ListView.GetHandler != nil {
		var err = views.Invoke(model.ListView.GetHandler(adminSite, app, model), w, r)
		except.AssertNil(err, 500, err)
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
		GetListFn: func(amount, offset uint) ([]attrs.Definer, error) {
			return model.GetListInstances(amount, offset)
		},
		TitleFieldColumn: func(lc list.ListColumn[attrs.Definer]) list.ListColumn[attrs.Definer] {
			return list.TitleFieldColumn(lc, func(defs attrs.Definitions, instance attrs.Definer) string {
				var primaryField = defs.Primary()
				if primaryField == nil {
					return ""
				}
				return django.Reverse("admin:apps:model:edit", app.Name, model.GetName(), primaryField.GetValue())
			})
		},
	}

	views.Invoke(view, w, r)
}

var ModelAddHandler = func(w http.ResponseWriter, r *http.Request, adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition) {

	if !permissions.HasObjectPermission(r, model.NewInstance(), "admin:add") {
		autherrors.Fail(
			http.StatusForbidden,
			"Permission denied",
		)
		return
	}

	var instance = model.NewInstance()
	if model.AddView.GetHandler != nil {
		var err = views.Invoke(model.AddView.GetHandler(adminSite, app, model, instance), w, r)
		except.AssertNil(err, 500, err)
		return
	}
	var addView = newInstanceView("add", instance, model.AddView, app, model, r)
	views.Invoke(addView, w, r)
	// if err := views.Invoke(addView, w, r); err != nil {
	// panic(err)
	// }
}

var ModelEditHandler = func(w http.ResponseWriter, r *http.Request, adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition, instance attrs.Definer) {
	if !permissions.HasObjectPermission(r, instance, "admin:edit") {
		autherrors.Fail(
			http.StatusForbidden,
			"Permission denied",
		)
		return
	}

	if model.EditView.GetHandler != nil {
		var err = views.Invoke(model.EditView.GetHandler(adminSite, app, model, instance), w, r)
		except.AssertNil(err, 500, err)
		return
	}
	var editView = newInstanceView("edit", instance, model.EditView, app, model, r)
	views.Invoke(editView, w, r)
	// if err := views.Invoke(editView, w, r); err != nil {
	// panic(err)
	// }
}

var ModelDeleteHandler = func(w http.ResponseWriter, r *http.Request, adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition, instance attrs.Definer) {
	if !permissions.HasObjectPermission(r, instance, "admin:delete") {
		autherrors.Fail(
			http.StatusForbidden,
			"Permission denied",
		)
		return
	}

	if model.DeleteView.GetHandler != nil {
		var err = views.Invoke(model.DeleteView.GetHandler(adminSite, app, model, instance), w, r)
		except.AssertNil(err, 500, err)
		return
	}

	var err error
	var context = NewContext(r, adminSite, nil)
	if r.Method == http.MethodPost {

		var hooks = goldcrest.Get[AdminModelHookFunc](
			AdminModelHookDelete,
		)
		for _, hook := range hooks {
			hook(r, AdminSite, model, instance)
		}

		if deleter, ok := instance.(models.Deleter); ok {
			err = deleter.Delete(r.Context())
		}
		if err != nil {
			context.Set("error", err)
		}

		var listViewURL = django.Reverse("admin:apps:model", app.Name, model.GetName())
		http.Redirect(w, r, listViewURL, http.StatusSeeOther)
		return
	}

	var definitions = instance.FieldDefs()
	var primaryField = definitions.Primary()

	context.Set("app", app)
	context.Set("model", model)
	context.Set("instance", instance)
	context.Set("primaryField", primaryField)

	err = tpl.FRender(w, context, BASE_KEY, "admin/views/models/delete.tmpl")
	except.AssertNil(err, 500, err)
}

func GetAdminForm(instance attrs.Definer, opts FormViewOptions, app *AppDefinition, model *ModelDefinition, r *http.Request) modelforms.ModelForm[attrs.Definer] {
	var form modelforms.ModelForm[attrs.Definer]
	if f, ok := instance.(FormDefiner); ok {
		form = f.AdminForm(r, app, model)
	} else {
		var modelForm = modelforms.NewBaseModelForm(instance)
		for _, field := range model.ModelFields(opts.ViewOptions, instance) {
			var (
				name      = field.Name()
				formfield = field.FormField()
			)

			var label = model.GetLabel(
				opts.ViewOptions,
				name,
				field.Label(),
			)

			formfield.SetLabel(label)

			modelForm.AddField(
				name, formfield,
			)
		}

		if opts.SaveInstance != nil {
			modelForm.SaveInstance = opts.SaveInstance
		} else {
			modelForm.SaveInstance = func(ctx context.Context, instance attrs.Definer) error {
				if saver, ok := instance.(models.Saver); ok {
					return saver.Save(ctx)
				}
				return errors.New("instance does not implement models.Saver")
			}
		}
		form = modelForm
	}

	return form
}

func newInstanceView(tpl string, instance attrs.Definer, opts FormViewOptions, app *AppDefinition, model *ModelDefinition, r *http.Request) *views.FormView[*AdminModelForm[modelforms.ModelForm[attrs.Definer]]] {

	var definer = instance.FieldDefs()
	var primary = definer.Primary()

	return &views.FormView[*AdminModelForm[modelforms.ModelForm[attrs.Definer]]]{
		BaseView: views.BaseView{
			AllowedMethods:  []string{http.MethodGet, http.MethodPost},
			BaseTemplateKey: BASE_KEY,
			TemplateName:    fmt.Sprintf("admin/views/models/%s.tmpl", tpl),
			GetContextFn: func(req *http.Request) (ctx.Context, error) {
				var context = NewContext(req, AdminSite, nil)
				context.Set("app", app)
				context.Set("model", model)
				context.Set("instance", instance)
				context.Set("primaryField", primary)
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

			for name, widget := range opts.Widgets {
				form.AddWidget(name, widget)
			}

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

			var adminFormFields = adminForm.Fields()

			if opts.Panels == nil {
				var panels = make([]Panel, 0, len(adminFormFields))
				for _, field := range adminFormFields {
					if _, ok := fields[field.Name()]; !ok {
						panels = append(panels, FieldPanel(field.Name()))
					}
				}
				adminForm.Panels = panels
			} else {
				for _, field := range adminFormFields {
					if _, ok := fields[field.Name()]; !ok {
						adminForm.DeleteField(field.Name())
					}
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

			var hooks = goldcrest.Get[AdminModelHookFunc](
				fmt.Sprintf("admin:model:%s", tpl),
			)
			for _, hook := range hooks {
				hook(req, AdminSite, model, instance)
			}

			except.AssertNotNil(instance, 500, "instance is nil after form submission")
			var listViewURL = django.Reverse("admin:apps:model", app.Name, model.GetName())
			http.Redirect(w, r, listViewURL, http.StatusSeeOther)
		},
	}
}
