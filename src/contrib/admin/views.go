package admin

import (
	"context"
	"fmt"
	"net/http"
	"slices"

	queries "github.com/Nigel2392/go-django/queries/src"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/admin/components"
	autherrors "github.com/Nigel2392/go-django/src/contrib/auth/auth_errors"
	"github.com/Nigel2392/go-django/src/contrib/filters"
	"github.com/Nigel2392/go-django/src/contrib/messages"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/core/pagination"
	"github.com/Nigel2392/go-django/src/core/trans"
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
			trans.T(r.Context(), "app %q does not have an index view"),
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

	if model.DisallowList {
		messages.Error(r, trans.T(r.Context(), "This model does not allow listing"))
		autherrors.Fail(
			http.StatusForbidden,
			trans.T(r.Context(), "This model does not allow listing"),
			django.Reverse("admin:home"),
		)
		return
	}

	var actions = make(map[string]BulkAction)
	var buttons = []components.ShowableComponent{
		components.NewShowableComponent(
			r,
			func(r *http.Request) bool {
				return len(actions) > 0
			},
			components.Button(components.ButtonConfig{
				Text: trans.GetTextFunc("Select All"),
				Type: components.ButtonTypePrimary,
				Attrs: map[string]any{
					"type":                     "button",
					"data-bulk-actions-target": "selectAll",
				},
			}),
		),
	}

	var hasBulkActionsPerm = permissions.HasPermission(r, "admin:bulk_actions")
	if len(model.ListView.BulkActions) > 0 && hasBulkActionsPerm {
		for _, action := range model.ListView.BulkActions {
			if !action.HasPermission(r, model) {
				continue
			}

			actions[action.Name()] = action
			buttons = append(buttons, components.NewShowableComponent(
				r,
				func(r *http.Request) bool {
					return action.HasPermission(r, model)
				},
				action.Button(),
			))
		}
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

	var qs *queries.QuerySet[attrs.Definer]
	if model.ListView.GetQuerySet != nil {
		qs = model.ListView.GetQuerySet(adminSite, app, model)
	} else {
		qs = queries.GetQuerySet(model.NewInstance())
	}
	if len(model.ListView.Prefetch.SelectRelated) > 0 {
		qs = qs.SelectRelated(model.ListView.Prefetch.SelectRelated...)
	}
	if len(model.ListView.Prefetch.PrefetchRelated) > 0 {
		qs = qs.Preload(model.ListView.Prefetch.PrefetchRelated...)
	}
	if len(model.ListView.Ordering) > 0 {
		qs = qs.OrderBy(model.ListView.Ordering...)
	}

	if !model.DisallowEdit && permissions.HasPermission(r, "admin:edit") {
		r = r.WithContext(list.SetAllowListEdit(r.Context(), true))
	}

	if len(actions) > 0 && hasBulkActionsPerm {
		r = r.WithContext(list.SetAllowListRowSelect(r.Context(), true))

		if r.Method == http.MethodPost {
			r.ParseForm()
			var listSelected = r.PostForm["list__row_select"]
			var actionName = r.PostFormValue("list_action")
			var meta = attrs.GetModelMeta(model.Model)
			var defs = meta.Definitions()
			var primaryField = defs.Primary()

			qs = qs.Filter(
				fmt.Sprintf("%s__in", primaryField.Name()),
				listSelected,
			)

			var action, ok = actions[actionName]
			if !ok {
				except.Fail(
					http.StatusBadRequest,
					trans.T(r.Context(), "Invalid list action"),
				)
				return
			}

			var changed, err = action.Execute(w, r, model, qs)
			if err != nil {
				logger.Errorf(
					"Failed to execute bulk action %s: %v",
					action.Name(), err,
				)
				except.Fail(
					http.StatusInternalServerError,
					trans.T(r.Context(), "Failed to apply list action"),
				)
				return
			}

			switch {
			case changed == 1:
				messages.Success(r, trans.T(r.Context(), "List action applied successfully to one object"))
				http.Redirect(w, r, django.Reverse("admin:apps:model", app.Name, model.GetName()), http.StatusSeeOther)
			case changed > 1:
				messages.Success(r, trans.T(r.Context(), "List action applied successfully to multiple objects"))
				http.Redirect(w, r, django.Reverse("admin:apps:model", app.Name, model.GetName()), http.StatusSeeOther)
			}

			return
		}
	}

	var filterForm *filters.Filters[attrs.Definer]
	if len(model.ListView.Filters) > 0 {
		filterForm = filters.NewFilters[attrs.Definer](r.Context(), "filters")
		for _, f := range model.ListView.Filters {
			filterForm.Add(f)
		}

		var err error
		qs, err = filterForm.Filter(r, r.URL.Query(), qs)
		if err != nil {
			except.AssertNil(err, 500, err)
			return
		}
	}

	var view = &list.View[attrs.Definer]{
		ListColumns:     columns,
		DefaultAmount:   int(amount),
		AllowedMethods:  []string{http.MethodGet, http.MethodPost},
		BaseTemplateKey: BASE_KEY,
		TemplateName:    "admin/views/models/list.tmpl",
		List: func(r *http.Request, po pagination.PageObject[attrs.Definer], lc []list.ListColumn[attrs.Definer], ctx ctx.Context) (list.StringRenderer, error) {
			if model.ListView.GetList != nil {
				return model.ListView.GetList(r, adminSite, app, model, po.Results())
			}
			return nil, nil
		},
		GetContextFn: func(req *http.Request, qs *queries.QuerySet[attrs.Definer]) (ctx.Context, error) {
			var paginator = list.PaginatorFromContext[attrs.Definer](req.Context())
			var count, err = paginator.Count()
			if err != nil {
				return nil, err
			}

			var context = NewContext(req, adminSite, nil)
			context.SetPage(PageOptions{
				TitleFn: trans.S(
					"%s List (%d)",
					model.Label(r.Context()),
					count,
				),
				Buttons: append(
					[]components.ShowableComponent{
						components.NewShowableComponent(
							req, func(r *http.Request) bool {
								return !model.DisallowCreate && permissions.HasObjectPermission(r, model.NewInstance(), "admin:add")
							},
							components.Link(components.ButtonConfig{
								Text: trans.S("Add"),
								Type: components.ButtonTypePrimary,
							}, func() string {
								return django.Reverse("admin:apps:model:add", app.Name, model.GetName())
							}),
						),
					},
					buttons...,
				),
			})
			context.Set("app", app)
			context.Set("model", model)
			context.Set("actions", actions)
			if filterForm != nil {
				context.Set("filter", filterForm)
			}
			return context, nil
		},
		QuerySet: func(r *http.Request) *queries.QuerySet[attrs.Definer] {
			return qs
		},
		TitleFieldColumn: func(lc list.ListColumn[attrs.Definer]) list.ListColumn[attrs.Definer] {
			var col = list.TitleFieldColumn(lc, func(r *http.Request, defs attrs.Definitions, instance attrs.Definer) string {
				if !permissions.HasObjectPermission(r, instance, "admin:edit") || model.DisallowEdit {
					return ""
				}

				var primaryField = defs.Primary()
				if primaryField == nil {
					return ""
				}
				return django.Reverse("admin:apps:model:edit", app.Name, model.GetName(), primaryField.GetValue())
			})
			if len(actions) > 0 {
				return list.RowSelectColumn(
					"list__row_select",
					nil,
					nil,
					col,
					map[string]any{
						"data-bulk-actions-target": "checkbox",
					},
				)
			}
			return col
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
		messages.Error(r, trans.T(r.Context(), "This model does not allow creation"))
		autherrors.Fail(
			http.StatusForbidden,
			trans.T(r.Context(), "This model does not allow creation"),
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

	var addView = newInstanceView("add", instance, model.AddView, app, model, r, &PageOptions{
		TitleFn: trans.S("Add %s", model.Label(r.Context())),
	})
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
		messages.Error(r, trans.T(r.Context(), "This model does not allow editing"))
		autherrors.Fail(
			http.StatusForbidden,
			trans.T(r.Context(), "This model does not allow editing"),
			django.Reverse("admin:apps:model", app.Name, model.GetName()),
		)
		return
	}

	if model.EditView.GetHandler != nil {
		var err = views.Invoke(model.EditView.GetHandler(adminSite, app, model, instance), w, r)
		except.AssertNil(err, 500, err)
		return
	}

	var editView = newInstanceView("edit", instance, model.EditView, app, model, r, &PageOptions{
		TitleFn: trans.S("Edit %s", model.Label(r.Context())),
	})
	views.Invoke(editView, w, r)
	// if err := views.Invoke(editView, w, r); err != nil {
	// panic(err)
	// }
}

type AdminDeleteView struct {
	views.BaseView
	Permissions []string
	Writer      http.ResponseWriter
	Request     *http.Request
	AdminSite   *AdminApplication
	App         *AppDefinition
	Model       *ModelDefinition
	Instances   []attrs.Definer
}

func (v *AdminDeleteView) Instance() attrs.Definer {
	if len(v.Instances) == 0 {
		except.Fail(
			http.StatusInternalServerError,
			"AdminDeleteView.Instance called with no model instances",
		)
	}
	return v.Instances[0]
}

func (v *AdminDeleteView) PrimaryField() attrs.FieldDefinition {
	var meta = attrs.GetModelMeta(v.Model.Model)
	var defs = meta.Definitions()
	return defs.Primary()
}

func (v *AdminDeleteView) PKList() []interface{} {
	var pks = make([]interface{}, len(v.Instances))
	for i, instance := range v.Instances {
		pks[i] = attrs.PrimaryKey(instance)
	}
	return pks
}

func (v *AdminDeleteView) Setup(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *http.Request) {
	if len(v.Permissions) > 0 && !permissions.HasObjectPermission(r, v.Instance(), v.Permissions...) {
		ReLogin(w, r, r.URL.Path)
		return nil, nil
	}

	if v.Model.DisallowDelete {
		messages.Error(r, trans.T(r.Context(), "This model does not allow deletion"))
		autherrors.Fail(
			http.StatusForbidden,
			trans.T(r.Context(), "This model does not allow deletion"),
			django.Reverse("admin:apps:model", v.App.Name, v.Model.GetName()),
		)
		return nil, nil
	}

	v.Writer = w
	v.Request = r

	return w, r
}

func (v *AdminDeleteView) ServePOST(w http.ResponseWriter, r *http.Request) {

	var hooks = goldcrest.Get[AdminModelDeleteFunc](
		AdminModelHookDelete,
	)
	for _, hook := range hooks {
		hook(r, AdminSite, v.Model, v.Instances)
	}

	var err error
	if v.Model.DeleteView.DeleteInstances != nil {
		err = v.Model.DeleteView.DeleteInstances(r.Context(), v.Instances)
	} else {
		var deleted int64
		deleted, err = queries.GetQuerySetWithContext(r.Context(), v.Model.NewInstance()).Delete(v.Instances...)
		assert.False(
			err == nil && deleted == 0,
			trans.T(r.Context(),
				"model %T not deleted, model does not implement models.Deleter interface", v.Instance,
			),
		)
	}

	var pks interface{}
	var pkList = v.PKList()
	if len(pkList) == 1 {
		pks = pkList[0]
	} else {
		pks = pkList
	}

	if err != nil {
		messages.Error(r,
			trans.T(r.Context(),
				"Failed to delete %s (%v): %v",
				v.Model._cType.Label(r.Context()),
				pks,
				err,
			),
		)
	} else {
		messages.Warning(r,
			trans.T(r.Context(),
				"Successfully deleted %s (%v)",
				v.Model._cType.Label(r.Context()),
				pks,
			),
		)
	}

	if nextURL := r.FormValue("next"); nextURL != "" {
		http.Redirect(w, r, nextURL, http.StatusSeeOther)
		return
	}

	var listViewURL = django.Reverse("admin:apps:model", v.App.Name, v.Model.GetName())
	http.Redirect(w, r, listViewURL, http.StatusSeeOther)
}

func (v *AdminDeleteView) GetContext(req *http.Request) (ctx.Context, error) {

	context := NewContext(req, v.AdminSite, nil)
	context.Set("app", v.App)
	context.Set("model", v.Model)
	context.Set("instances", v.Instances)
	context.Set("pk_list", v.PKList())

	if len(v.Instances) == 1 {
		context.Set("instance", v.Instance())
		context.Set("primaryField", v.PrimaryField())
	}

	var next = req.FormValue("next")
	if next != "" {
		context.Set("BackURL", next)
	}

	return context, nil
}

var ModelDeleteHandler = func(w http.ResponseWriter, r *http.Request, adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition, instance attrs.Definer) {
	if model.DeleteView.GetHandler != nil {
		var err = views.Invoke(model.DeleteView.GetHandler(adminSite, app, model, []attrs.Definer{instance}), w, r)
		except.AssertNil(err, 500, err)
		return
	}

	var deleteView = &AdminDeleteView{
		BaseView: views.BaseView{
			AllowedMethods:  []string{http.MethodGet, http.MethodPost},
			BaseTemplateKey: BASE_KEY,
			TemplateName:    "admin/views/models/delete.tmpl",
		},
		Permissions: []string{"admin:delete"},
		Writer:      w,
		Request:     r,
		AdminSite:   adminSite,
		App:         app,
		Model:       model,
		Instances:   []attrs.Definer{instance},
	}

	var err = views.Invoke(deleteView, w, r)
	except.AssertNil(err, 500, err)
}

var ModelBulkDeleteHandler = func(w http.ResponseWriter, r *http.Request, adminSite *AdminApplication, app *AppDefinition, model *ModelDefinition) {

	var pkList = r.URL.Query()["pk_list"]
	if len(pkList) == 0 {
		except.Fail(
			http.StatusBadRequest,
			trans.T(r.Context(), "No primary keys provided for bulk delete"),
		)
		return
	}

	var meta = attrs.GetModelMeta(model.Model)
	var defs = meta.Definitions()
	var prim = defs.Primary()
	var instanceRows, err = queries.GetQuerySetWithContext(r.Context(), model.NewInstance()).
		Filter(fmt.Sprintf("%s__in", prim.Name()), pkList).
		SelectRelated(model.ListView.Prefetch.SelectRelated...).
		Preload(model.ListView.Prefetch.PrefetchRelated...).
		All()

	if err != nil {
		except.Fail(
			http.StatusInternalServerError,
			trans.T(r.Context(),
				"Failed to retrieve instances for bulk delete: %v",
				err,
			),
		)
		return
	}

	var instances = slices.Collect(instanceRows.Objects())
	if model.DeleteView.GetHandler != nil {
		var err = views.Invoke(model.DeleteView.GetHandler(adminSite, app, model, instances), w, r)
		except.AssertNil(err, 500, err)
		return
	}

	var deleteView = &AdminDeleteView{
		BaseView: views.BaseView{
			AllowedMethods:  []string{http.MethodGet, http.MethodPost},
			BaseTemplateKey: BASE_KEY,
			TemplateName:    "admin/views/models/delete.tmpl",
		},
		Permissions: []string{"admin:delete"},
		Writer:      w,
		Request:     r,
		AdminSite:   adminSite,
		App:         app,
		Model:       model,
		Instances:   instances,
	}

	err = views.Invoke(deleteView, w, r)
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

				return except.Assert(
					saved, http.StatusInternalServerError,
					trans.T(r.Context(),
						"model %T not saved, model does not implement models.Saver interface",
						instance,
					),
				)
			}
		}
		form = modelForm
	}

	return form
}

func newInstanceView(tpl string, instance attrs.Definer, opts FormViewOptions, app *AppDefinition, model *ModelDefinition, r *http.Request, page *PageOptions) *views.FormView[*AdminModelForm[modelforms.ModelForm[attrs.Definer], attrs.Definer]] {

	var definer = instance.FieldDefs()
	var primary = definer.Primary()

	return &views.FormView[*AdminModelForm[modelforms.ModelForm[attrs.Definer], attrs.Definer]]{
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

				if page != nil {
					context.SetPage(*page)
				}

				var next = r.URL.Query().Get("next")
				if next != "" {
					context.Set("BackURL", next)
				}

				return context, nil
			},
		},
		GetFormFn: func(req *http.Request) *AdminModelForm[modelforms.ModelForm[attrs.Definer], attrs.Definer] {
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

			var adminForm = &AdminModelForm[modelforms.ModelForm[attrs.Definer], attrs.Definer]{
				AdminForm: &AdminForm[modelforms.ModelForm[attrs.Definer]]{
					Form:   form,
					Panels: opts.Panels,
				},
			}

			adminForm.Load()

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
		SuccessFn: func(w http.ResponseWriter, req *http.Request, form *AdminModelForm[modelforms.ModelForm[attrs.Definer], attrs.Definer]) {
			var instance = form.Instance()

			var hooks = goldcrest.Get[AdminModelHookFunc](
				fmt.Sprintf("admin:model:%s", tpl),
			)
			for _, hook := range hooks {
				hook(req, AdminSite, model, instance)
			}

			except.AssertNotNil(instance, 500, trans.T(
				r.Context(), "instance is nil after form submission",
			))

			messages.Success(req,
				trans.T(r.Context(),
					"Successfully saved %s (%v)",
					attrs.ToString(instance),
					attrs.PrimaryKey(instance),
				),
			)

			if nextURL := req.FormValue("next"); nextURL != "" {
				http.Redirect(w, req, nextURL, http.StatusSeeOther)
				return
			}

			var listViewURL = django.Reverse("admin:apps:model", app.Name, model.GetName())
			http.Redirect(w, r, listViewURL, http.StatusSeeOther)
		},
	}
}
