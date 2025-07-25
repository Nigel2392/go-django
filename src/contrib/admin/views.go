package admin

import (
	"context"
	"fmt"
	"net/http"
	"slices"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	django "github.com/Nigel2392/go-django/src"
	autherrors "github.com/Nigel2392/go-django/src/contrib/auth/auth_errors"
	"github.com/Nigel2392/go-django/src/contrib/messages"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
	"github.com/Nigel2392/go-django/src/models"
	"github.com/Nigel2392/go-django/src/permissions"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/go-django/src/views/list"
	"github.com/Nigel2392/goldcrest"
)

var AppHandler = func(w http.ResponseWriter, r *http.Request, adminSite *AdminApplication, app *AppDefinition) {
	if !app.Options.EnableIndexView {
		except.RaiseNotFound(
			"app %q does not have an index view",
			app.Label(r.Context()),
		)
		return
	}

	if !permissions.HasPermission(r, fmt.Sprintf("admin:view_app:%s", app.Name)) {
		ReLogin(w, r, r.URL.Path)
		return
	}

	if app.Options.IndexView != nil {
		var err = views.Invoke(app.Options.IndexView(adminSite, app), w, r)
		except.AssertNil(err, 500, err)
		return
	}

	var models = app.Models
	var modelNames = make([]string, models.Len())
	var i = 0
	for front := models.Front(); front != nil; front = front.Next() {
		modelNames[i] = front.Value.Label(r.Context())
		i++
	}

	var hookName = RegisterAdminPageComponentHook(app.Name)
	var hook = goldcrest.Get[RegisterAdminAppPageComponentHookFunc](hookName)
	var components = make([]AdminPageComponent, 0)
	for _, h := range hook {
		var component = h(r, adminSite, app)
		if component != nil {
			components = append(components, component)
		}
	}
	sortComponents(components)

	var context = NewContext(
		r, adminSite, nil,
	)

	context.SetPage(PageOptions{
		TitleFn:    app.Label,
		SubtitleFn: app.Description,
		MediaFn: func() media.Media {
			var appMedia media.Media = media.NewMedia()
			for _, component := range components {
				var m = component.Media()
				if m != nil {
					appMedia = appMedia.Merge(m)
				}
			}
			return appMedia
		},
	})
	context.Set("app", app)
	context.Set("models", modelNames)
	context.Set("components", components)
	var err = tpl.FRender(w, context, BASE_KEY, "admin/views/apps/index.tmpl")
	except.AssertNil(err, 500, err)
}

var ModelListHandler = func(w http.ResponseWriter, r *http.Request, adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition) {

	if !permissions.HasObjectPermission(r, model.NewInstance(), "admin:view_list") {
		ReLogin(w, r, r.URL.Path)
		return
	}

	if model.ListView.GetHandler != nil {
		var err = views.Invoke(model.ListView.GetHandler(adminSite, app, model), w, r)
		except.AssertNil(err, 500, err)
		return
	}

	var columns = make([]list.ListColumn[attrs.Definer], len(model.ListView.Fields))
	for i, field := range model.ListView.Fields {
		columns[i] = model.GetColumn(r.Context(), model.ListView, field)
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
			if model.ListView.GetQuerySet != nil {
				var qs = model.ListView.GetQuerySet(adminSite, app, model).
					WithContext(r.Context()).
					Offset(int(offset)).
					Limit(int(amount))

				var rows, err = qs.All()
				if err != nil {
					return nil, err
				}

				return slices.Collect(rows.Objects()), nil
			}

			return model.GetListInstances(r.Context(), amount, offset)
		},
		TitleFieldColumn: func(lc list.ListColumn[attrs.Definer]) list.ListColumn[attrs.Definer] {
			return list.TitleFieldColumn(lc, func(r *http.Request, defs attrs.Definitions, instance attrs.Definer) string {
				if !permissions.HasObjectPermission(r, instance, "admin:edit") || model.DisallowEdit {
					return ""
				}

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
		ReLogin(w, r, r.URL.Path)
		return
	}

	if model.DisallowCreate {
		messages.Error(r, "This model does not allow creation")
		autherrors.Fail(
			http.StatusForbidden,
			"This model does not allow creation",
			django.Reverse("admin:apps:model", app.Name, model.GetName()),
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
		ReLogin(w, r, r.URL.Path)
		return
	}

	if model.DisallowEdit {
		messages.Error(r, "This model does not allow editing")
		autherrors.Fail(
			http.StatusForbidden,
			"This model does not allow editing",
			django.Reverse("admin:apps:model", app.Name, model.GetName()),
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
		ReLogin(w, r, r.URL.Path)
		return
	}

	if model.DisallowDelete {
		messages.Error(r, "This model does not allow deletion")
		autherrors.Fail(
			http.StatusForbidden,
			"This model does not allow deletion",
			django.Reverse("admin:apps:model", app.Name, model.GetName()),
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

		if model.DeleteView.DeleteInstance != nil {
			err = model.DeleteView.DeleteInstance(r.Context(), instance)
		} else {
			var deleted bool
			deleted, err = models.DeleteModel(r.Context(), instance)
			assert.False(
				err == nil && !deleted,
				"model %T not deleted, model does not implement models.Deleter interface",
				instance,
			)
		}

		if err != nil {
			context.Set("error", err)

			var asErr = &errors.Error{Code: errors.CodeCheckFailed}
			if errors.As(err, asErr) {
				err = asErr.Wrapf(
					"failed to delete %s (%v)",
					attrs.ToString(instance),
					attrs.PrimaryKey(instance),
				)
			}

			messages.Error(r,
				fmt.Sprintf(
					"Failed to delete %s (%v): %v",
					attrs.ToString(instance),
					attrs.PrimaryKey(instance),
					err,
				),
			)
		} else {
			messages.Warning(r,
				fmt.Sprintf(
					"Successfully deleted %s (%v)",
					attrs.ToString(instance),
					attrs.PrimaryKey(instance),
				),
			)
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
		var modelForm = modelforms.NewBaseModelForm(r.Context(), instance)
		for _, field := range model.ModelFields(opts.ViewOptions, instance) {
			var (
				name      = field.Name()
				formfield = field.FormField()
			)

			if formfield == nil {
				logger.Warnf(
					"Field %q for model %T does not have a form field",
					name, instance,
				)
				continue
			}

			var label = model.GetLabel(
				opts.ViewOptions,
				name,
				field.Label(r.Context()),
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
				var saved, err = models.SaveModel(ctx, instance)
				if err != nil {
					return err
				}

				return assert.True(
					saved,
					"model %T not saved, model does not implement models.Saver interface",
					instance,
				)
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
				form = GetAdminForm(instance, opts, app, model, req)
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
				if attrs.IsZero(v) {
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

			messages.Success(req,
				fmt.Sprintf(
					"Successfully saved %s (%v)",
					attrs.ToString(instance),
					attrs.PrimaryKey(instance),
				),
			)

			var listViewURL = django.Reverse("admin:apps:model", app.Name, model.GetName())
			http.Redirect(w, r, listViewURL, http.StatusSeeOther)
		},
	}
}
