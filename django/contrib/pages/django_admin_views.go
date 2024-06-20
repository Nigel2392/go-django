package pages

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Nigel2392/django"
	"github.com/Nigel2392/django/contrib/admin"
	"github.com/Nigel2392/django/contrib/auth"
	"github.com/Nigel2392/django/contrib/pages/models"
	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/core/contenttypes"
	"github.com/Nigel2392/django/core/ctx"
	"github.com/Nigel2392/django/core/except"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/django/forms/modelforms"
	"github.com/Nigel2392/django/permissions"
	"github.com/Nigel2392/django/views"
	"github.com/Nigel2392/django/views/list"
	"github.com/Nigel2392/mux"
)

func pageHandler(fn func(http.ResponseWriter, *http.Request, *admin.AppDefinition, *admin.ModelDefinition, *models.PageNode)) mux.Handler {
	return mux.NewHandler(func(w http.ResponseWriter, req *http.Request) {

		var (
			ctx       = req.Context()
			routeVars = mux.Vars(req)
			pageID    = routeVars.GetInt("page_id")
		)
		if pageID == 0 {
			http.Error(w, "invalid page id", http.StatusBadRequest)
			return
		}

		var page, err = QuerySet().GetNodeByID(ctx, int64(pageID))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var app, ok = admin.AdminSite.Apps.Get(AdminPagesAppName)
		if !ok {
			http.Error(w, "App not found", http.StatusNotFound)
			return
		}

		model, ok := app.Models.Get(AdminPagesModelPath)
		if !ok {
			http.Error(w, "Model not found", http.StatusNotFound)
			return
		}

		fn(w, req, app, model, &page)
	})
}

func listPageHandler(w http.ResponseWriter, r *http.Request, a *admin.AppDefinition, m *admin.ModelDefinition, p *models.PageNode) {

	if !permissions.Object(r, "pages:list", p) {
		auth.Fail(
			http.StatusForbidden,
			"User does not have permission to view this page",
		)
		return
	}

	var columns = make([]list.ListColumn[attrs.Definer], len(m.ListView.Fields)+1)
	for i, field := range m.ListView.Fields {
		columns[i+1] = m.GetColumn(m.ListView, field)
	}

	columns[0] = columns[1]
	columns[1] = &admin.ListActionsColumn[attrs.Definer]{
		Actions: []*admin.ListAction[attrs.Definer]{
			{
				Text: func(defs attrs.Definitions, row attrs.Definer) string {
					return fields.T("Edit Page")
				},
				URL: func(defs attrs.Definitions, row attrs.Definer) string {
					var primaryField = defs.Primary()
					if primaryField == nil {
						return ""
					}
					return django.Reverse(
						"admin:pages:edit",
						primaryField.GetValue(),
					)
				},
			},
			{
				Show: func(defs attrs.Definitions, row attrs.Definer) bool {
					return row.(*models.PageNode).Numchild > 0
				},
				Text: func(defs attrs.Definitions, row attrs.Definer) string {
					return fields.T("Add Child")
				},
				URL: func(defs attrs.Definitions, row attrs.Definer) string {
					var primaryField = defs.Primary()
					if primaryField == nil {
						return ""
					}
					return django.Reverse(
						"admin:pages:type",
						primaryField.GetValue(),
					)
				},
			},
			//{
			//	isShown: func(defs attrs.Definitions, row attrs.Definer) bool {
			//		return row.(*models.PageNode).Numchild > 0
			//	},
			//	getText: func(defs attrs.Definitions, row attrs.Definer) string {
			//		return fields.T("View Children")
			//	},
			//	getURL: func(defs attrs.Definitions, row attrs.Definer) string {
			//		var primaryField = defs.Primary()
			//		if primaryField == nil {
			//			return ""
			//		}
			//		return django.Reverse(
			//			"admin:pages:list",
			//			primaryField.GetValue(),
			//		)
			//	},
			//},
		},
	}

	var amount = m.ListView.PerPage
	if amount == 0 {
		amount = 25
	}

	var parent_object *models.PageNode
	if p.Depth > 0 {
		var parent, err = ParentNode(
			QuerySet(),
			r.Context(),
			p.Path,
			int(p.Depth),
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		parent_object = &parent
	}

	var view = &list.View[attrs.Definer]{
		ListColumns:   columns,
		DefaultAmount: amount,
		BaseView: views.BaseView{
			AllowedMethods:  []string{http.MethodGet, http.MethodPost},
			BaseTemplateKey: admin.BASE_KEY,
			TemplateName:    "pages/admin/admin_list.tmpl",
			GetContextFn: func(req *http.Request) (ctx.Context, error) {
				var context = admin.NewContext(req, admin.AdminSite, nil)
				context.Set("app", a)
				context.Set("model", m)
				context.Set("page_object", p)
				context.Set("parent_object", parent_object)
				return context, nil
			},
		},
		GetListFn: func(amount, offset uint, include []string) ([]attrs.Definer, error) {
			var ctx = r.Context()
			var nodes, err = QuerySet().GetChildNodes(ctx, p.Path, p.Depth)
			if err != nil {
				return nil, err
			}
			var items = make([]attrs.Definer, 0, len(nodes))
			for _, n := range nodes {
				n := n
				items = append(items, &n)
			}
			return items, nil
		},
		TitleFieldColumn: func(lc list.ListColumn[attrs.Definer]) list.ListColumn[attrs.Definer] {
			return list.TitleFieldColumn(lc, func(defs attrs.Definitions, instance attrs.Definer) string {
				var primaryField = defs.Primary()
				if primaryField == nil {
					return ""
				}
				return django.Reverse("admin:pages:edit", primaryField.GetValue())
			})
		},
	}

	views.Invoke(view, w, r)
}

func choosePageTypeHandler(w http.ResponseWriter, r *http.Request, a *admin.AppDefinition, m *admin.ModelDefinition, p *models.PageNode) {

	if !permissions.Object(r, "pages:add", p) {
		auth.Fail(
			http.StatusForbidden,
			"User does not have permission to add a page",
		)
		return
	}

	var definitions = ListDefinitions()
	var view = &views.BaseView{
		AllowedMethods:  []string{http.MethodGet},
		BaseTemplateKey: admin.BASE_KEY,
		TemplateName:    "pages/admin/choose_page_type.tmpl",
		GetContextFn: func(req *http.Request) (ctx.Context, error) {
			var context = admin.NewContext(req, admin.AdminSite, nil)
			context.Set("app", a)
			context.Set("model", m)
			context.Set("page_object", p)
			context.Set("definitions", definitions)
			context.SetPage(admin.PageOptions{
				TitleFn:    fields.S("Choose Page Type"),
				SubtitleFn: fields.S("Select the type of page you want to create"),
			})
			return context, nil
		},
	}

	views.Invoke(view, w, r)
}

func addPageHandler(w http.ResponseWriter, r *http.Request, a *admin.AppDefinition, m *admin.ModelDefinition, p *models.PageNode) {
	if !permissions.Object(r, "pages:add", p) {
		auth.Fail(
			http.StatusForbidden,
			"User does not have permission to add a page",
		)
		return
	}

	fmt.Println("Add page handler")

	var cType = contenttypes.DefinitionForPackage("core", "BlogPage")
	var page pageDefiner
	if cType != nil {
		page = cType.Object().(pageDefiner)
	} else {
		page = p
	}

	fmt.Println("Page: ", page, cType)

	var fieldDefs = page.FieldDefs()
	var definition = DefinitionForObject(page)
	var panels []admin.Panel
	if definition == nil || definition.AddPanels == nil {
		panels = make([]admin.Panel, fieldDefs.Len())

		for i, def := range fieldDefs.Fields() {
			panels[i] = admin.FieldPanel(def.Name())
		}
	} else {
		panels = definition.AddPanels(r, page)
	}

	var form = modelforms.NewBaseModelForm[attrs.Definer](page)
	var adminForm = admin.NewAdminModelForm[modelforms.ModelForm[attrs.Definer]](
		form, panels...,
	)

	adminForm.Load()

	form.SaveInstance = func(ctx context.Context, d attrs.Definer) error {
		if page, ok := d.(SaveablePage); ok {
			return SavePage(QuerySet(), ctx, p, page)
		}

		var n = d.(*models.PageNode)
		var _, err = QuerySet().InsertNode(
			ctx, n.Title, n.Path, n.Depth, n.Numchild, n.UrlPath, int64(n.StatusFlags), n.PageID, n.ContentType,
		)
		return err
	}

	var view = &views.FormView[*admin.AdminModelForm[modelforms.ModelForm[attrs.Definer]]]{
		BaseView: views.BaseView{
			AllowedMethods:  []string{http.MethodGet, http.MethodPost},
			BaseTemplateKey: admin.BASE_KEY,
			TemplateName:    "pages/admin/add_page.tmpl",
			GetContextFn: func(req *http.Request) (ctx.Context, error) {
				var context = admin.NewContext(req, admin.AdminSite, nil)
				context.Set("app", a)
				context.Set("model", m)
				context.Set("page_object", p)
				context.Set("PostURL", django.Reverse("admin:pages:add", p.ID()))
				return context, nil
			},
		},
		GetFormFn: func(req *http.Request) *admin.AdminModelForm[modelforms.ModelForm[attrs.Definer]] {
			return adminForm
		},
		GetInitialFn: func(req *http.Request) map[string]interface{} {
			var initial = make(map[string]interface{})
			for _, field := range fieldDefs.Fields() {
				initial[field.Name()] = field.GetValue()
			}
			return initial
		},
		SuccessFn: func(w http.ResponseWriter, req *http.Request, form *admin.AdminModelForm[modelforms.ModelForm[attrs.Definer]]) {
			var instance = form.Instance()
			assert.False(instance == nil, "instance is nil after form submission")
			fmt.Println("Instance: ", instance, instance.(Page).Reference())
			var f = instance.(pageDefiner)
			var primary = f.FieldDefs().Primary()

			fmt.Println("Primary: ", primary.GetValue())

			var listViewURL = django.Reverse("admin:pages:list", primary.GetValue())
			http.Redirect(w, r, listViewURL, http.StatusSeeOther)
		},
	}

	if err := views.Invoke(view, w, r); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type pageDefiner interface {
	attrs.Definer
	Page
}

func editPageHandler(w http.ResponseWriter, r *http.Request, a *admin.AppDefinition, m *admin.ModelDefinition, p *models.PageNode) {

	if !permissions.Object(r, "pages:edit", p) {
		auth.Fail(
			http.StatusForbidden,
			"User does not have permission to edit this page",
		)
		return
	}

	var instance, err = Specific(r.Context(), *p)
	except.Assert(err == nil, 500, "error getting page instance, it might not exist.")

	var page, ok = instance.(pageDefiner)
	if !ok {
		page = p
		// logger.Fatalf(1, "instance does not adhere to attrs.Definer: %T", instance)
	}

	var fieldDefs = page.FieldDefs()
	var definition = DefinitionForObject(page)
	var panels []admin.Panel
	if definition == nil || definition.EditPanels == nil {
		panels = make([]admin.Panel, fieldDefs.Len())

		for i, def := range fieldDefs.Fields() {
			panels[i] = admin.FieldPanel(def.Name())
		}
	} else {
		panels = definition.EditPanels(r, page)
	}

	var form = modelforms.NewBaseModelForm[attrs.Definer](page)
	var adminForm = admin.NewAdminModelForm[modelforms.ModelForm[attrs.Definer]](form, panels...)

	adminForm.Load()

	form.SaveInstance = func(ctx context.Context, d attrs.Definer) error {
		if page, ok := d.(SaveablePage); ok {

			return UpdatePage(QuerySet(), ctx, page)
		}

		var n = d.(*models.PageNode)
		return QuerySet().UpdateNode(ctx, n.Title, n.Path, n.Depth, n.Numchild, n.UrlPath, int64(n.StatusFlags), n.PageID, n.ContentType, n.PK)
	}

	var view = &views.FormView[*admin.AdminModelForm[modelforms.ModelForm[attrs.Definer]]]{
		BaseView: views.BaseView{
			AllowedMethods:  []string{http.MethodGet, http.MethodPost},
			BaseTemplateKey: admin.BASE_KEY,
			TemplateName:    "pages/admin/edit_page.tmpl",
			GetContextFn: func(req *http.Request) (ctx.Context, error) {
				var context = admin.NewContext(req, admin.AdminSite, nil)
				var primary = fieldDefs.Primary()
				context.Set("app", a)
				context.Set("model", m)
				context.Set("page_object", page)
				context.Set("PostURL", django.Reverse("admin:pages:edit", primary.GetValue()))
				return context, nil
			},
		},
		GetFormFn: func(req *http.Request) *admin.AdminModelForm[modelforms.ModelForm[attrs.Definer]] {
			return adminForm
		},
		GetInitialFn: func(req *http.Request) map[string]interface{} {
			var initial = make(map[string]interface{})
			for _, field := range fieldDefs.Fields() {
				initial[field.Name()] = field.GetValue()
			}
			return initial
		},
		SuccessFn: func(w http.ResponseWriter, req *http.Request, form *admin.AdminModelForm[modelforms.ModelForm[attrs.Definer]]) {
			var instance = form.Instance()
			assert.False(instance == nil, "instance is nil after form submission")
			var listViewURL = django.Reverse("admin:pages:list", instance.(Page).ID())
			http.Redirect(w, r, listViewURL, http.StatusSeeOther)
		},
	}

	if err := views.Invoke(view, w, r); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
