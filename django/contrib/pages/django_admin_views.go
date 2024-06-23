package pages

import (
	"context"
	"net/http"
	"net/url"
	"slices"

	"github.com/Nigel2392/django"
	"github.com/Nigel2392/django/contrib/admin"
	"github.com/Nigel2392/django/contrib/auth"
	"github.com/Nigel2392/django/contrib/pages/models"
	auditlogs "github.com/Nigel2392/django/contrib/reports/audit_logs"
	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/core/contenttypes"
	"github.com/Nigel2392/django/core/ctx"
	"github.com/Nigel2392/django/core/except"
	"github.com/Nigel2392/django/core/logger"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/django/forms/modelforms"
	"github.com/Nigel2392/django/permissions"
	"github.com/Nigel2392/django/views"
	"github.com/Nigel2392/django/views/list"
	"github.com/Nigel2392/mux"
)

type pageDefiner interface {
	attrs.Definer
	Page
}

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

func getPageBreadcrumbs(r *http.Request, p *models.PageNode, urlForLast bool) ([]admin.BreadCrumb, error) {
	var breadcrumbs = make([]admin.BreadCrumb, 0, p.Depth+2)
	if p.Depth > 0 {
		var ancestors, err = AncestorNodes(
			QuerySet(), r.Context(), p.Path, int(p.Depth)+1,
		)
		if err != nil {
			return nil, err
		}
		slices.SortStableFunc(ancestors, func(a, b models.PageNode) int {
			if a.Depth < b.Depth {
				return -1
			}
			if a.Depth > b.Depth {
				return 1
			}
			return 0
		})

		for _, a := range ancestors {
			a := a
			breadcrumbs = append(breadcrumbs, admin.BreadCrumb{
				Title: a.Title,
				URL:   django.Reverse("admin:pages:list", a.ID()),
			})
		}

		var b = admin.BreadCrumb{
			Title: p.Title,
		}

		if urlForLast {
			b.URL = django.Reverse("admin:pages:list", p.ID())
		}

		breadcrumbs = append(breadcrumbs, b)
	}
	return breadcrumbs, nil
}

func listPageHandler(w http.ResponseWriter, r *http.Request, a *admin.AppDefinition, m *admin.ModelDefinition, p *models.PageNode) {

	if !permissions.HasObjectPermission(r, p, "pages:list") {
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

	var next = django.Reverse(
		"admin:pages:list",
		p.ID(),
	)

	columns[0] = columns[1]
	columns[1] = &admin.ListActionsColumn[attrs.Definer]{
		Actions: []*admin.ListAction[attrs.Definer]{
			{
				Show: func(defs attrs.Definitions, row attrs.Definer) bool {
					return permissions.HasObjectPermission(r, row, "pages:edit")
				},
				Text: func(defs attrs.Definitions, row attrs.Definer) string {
					return fields.T("Edit Page")
				},
				URL: func(defs attrs.Definitions, row attrs.Definer) string {
					var primaryField = defs.Primary()
					if primaryField == nil {
						return ""
					}
					var u = django.Reverse(
						"admin:pages:edit",
						primaryField.GetValue(),
					)
					var url, err = url.Parse(u)
					if err != nil {
						return u
					}
					var q = url.Query()
					q.Set("next", next)
					url.RawQuery = q.Encode()
					return url.String()
				},
			},
			{
				Show: func(defs attrs.Definitions, row attrs.Definer) bool {
					// return row.(*models.PageNode).Numchild > 0
					return permissions.HasObjectPermission(r, row, "pages:add")
				},
				Text: func(defs attrs.Definitions, row attrs.Definer) string {
					return fields.T("Add Child")
				},
				URL: func(defs attrs.Definitions, row attrs.Definer) string {
					var primaryField = defs.Primary()
					if primaryField == nil {
						return ""
					}
					var u = django.Reverse(
						"admin:pages:type",
						primaryField.GetValue(),
					)
					var url, err = url.Parse(u)
					if err != nil {
						return u
					}
					var q = url.Query()
					q.Set("next", next)
					url.RawQuery = q.Encode()
					return url.String()
				},
			},
			{
				Show: func(defs attrs.Definitions, row attrs.Definer) bool {
					return row.(*models.PageNode).Numchild > 0 && permissions.HasObjectPermission(
						r, row, "pages:list",
					)
				},
				Text: func(defs attrs.Definitions, row attrs.Definer) string {
					return fields.T("View Children")
				},
				URL: func(defs attrs.Definitions, row attrs.Definer) string {
					var primaryField = defs.Primary()
					if primaryField == nil {
						return ""
					}
					return django.Reverse(
						"admin:pages:list",
						primaryField.GetValue(),
					)
				},
			},
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
				var context = admin.NewContext(
					req, admin.AdminSite, nil,
				)

				var contentType *contenttypes.ContentTypeDefinition
				if p.ContentType != "" {
					contentType = contenttypes.DefinitionForType(
						p.ContentType,
					)
				} else {
					contentType = contenttypes.DefinitionForObject(
						p,
					)
				}

				context.Set("app", a)
				context.Set("model", m)
				context.Set("page_object", p)
				context.Set("parent_object", parent_object)
				context.Set(
					"model_name",
					contentType.Label(),
				)

				var breadcrumbs, err = getPageBreadcrumbs(r, p, false)
				if err != nil {
					return nil, err
				}
				context.SetPage(admin.PageOptions{
					BreadCrumbs: breadcrumbs,
				})
				return context, nil
			},
		},
		GetListFn: func(amount, offset uint, include []string) ([]attrs.Definer, error) {
			var ctx = r.Context()
			var nodes, err = QuerySet().GetChildNodes(ctx, p.Path, p.Depth, int32(amount), int32(offset))
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

				var u = django.Reverse(
					"admin:pages:edit",
					primaryField.GetValue(),
				)

				var url, err = url.Parse(u)
				if err != nil {
					return u
				}

				var q = url.Query()
				q.Set("next", next)
				url.RawQuery = q.Encode()
				return url.String()
			})
		},
	}

	views.Invoke(view, w, r)
}

func choosePageTypeHandler(w http.ResponseWriter, r *http.Request, a *admin.AppDefinition, m *admin.ModelDefinition, p *models.PageNode) {

	if !permissions.HasObjectPermission(r, p, "pages:add") {
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

			var next = req.URL.Query().Get("next")
			if next != "" {
				context.Set("BackURL", next)
			}

			var breadcrumbs, err = getPageBreadcrumbs(r, p, true)
			if err != nil {
				return nil, err
			}

			context.SetPage(admin.PageOptions{
				TitleFn:     fields.S("Choose Page Type"),
				SubtitleFn:  fields.S("Select the type of page you want to create"),
				BreadCrumbs: breadcrumbs,
			})

			return context, nil
		},
	}

	views.Invoke(view, w, r)
}

func addPageHandler(w http.ResponseWriter, r *http.Request, a *admin.AppDefinition, m *admin.ModelDefinition, p *models.PageNode) {
	if !permissions.HasObjectPermission(r, p, "pages:add") {
		auth.Fail(
			http.StatusForbidden,
			"User does not have permission to add a page",
		)
		return
	}

	var (
		vars       = mux.Vars(r)
		app_label  = vars.Get("app_label")
		model_name = vars.Get("model_name")
	)

	if app_label == "" || model_name == "" {
		http.Error(w, "app_label and model_name are required", http.StatusBadRequest)
		return
	}

	var cTypeDef = contenttypes.DefinitionForPackage(app_label, model_name)
	if cTypeDef == nil {
		http.Error(w, "content type not found", http.StatusNotFound)
		return
	}

	var (
		page       = cTypeDef.Object().(pageDefiner)
		cType      = cTypeDef.ContentType()
		fieldDefs  = page.FieldDefs()
		definition = DefinitionForObject(page)
		panels     []admin.Panel
	)
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

	form.SaveInstance = func(ctx context.Context, d attrs.Definer) (err error) {
		if page, ok := d.(SaveablePage); ok {
			err = SavePage(QuerySet(), ctx, p, page)
		} else {
			var n = d.(*models.PageNode)
			_, err = QuerySet().InsertNode(
				ctx, n.Title, n.Path, n.Depth, n.Numchild, n.UrlPath, n.Slug, int64(n.StatusFlags), n.PageID, n.ContentType,
			)
		}
		if err != nil {
			return err
		}

		auditlogs.Log("pages:add", logger.INF, page.Reference(), map[string]interface{}{
			"parent": p.ID(),
			"label":  page.Reference().Title,
			"cType":  cType.PkgPath(),
		})

		return django.Task("[TRANSACTION] Fixing tree structure upon manual page node save", func(app *django.Application) error {
			return FixTree(pageApp.QuerySet(), ctx)
		})
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

				var backURL string
				if q := req.URL.Query().Get("next"); q != "" {
					backURL = q
				}
				context.Set("BackURL", backURL)
				context.Set("PostURL", django.Reverse(
					"admin:pages:add",
					p.ID(),
					cType.AppLabel(),
					cType.Model(),
				))

				var breadcrumbs, err = getPageBreadcrumbs(r, p, true)
				if err != nil {
					return nil, err
				}

				context.SetPage(admin.PageOptions{
					TitleFn:     fields.S("Add %q", cType.Model()),
					BreadCrumbs: breadcrumbs,
				})

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
			var ref = instance.(Page).Reference()
			var listViewURL = django.Reverse("admin:pages:list", ref.ID())
			http.Redirect(w, r, listViewURL, http.StatusSeeOther)
		},
	}

	if err := views.Invoke(view, w, r); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func editPageHandler(w http.ResponseWriter, r *http.Request, a *admin.AppDefinition, m *admin.ModelDefinition, p *models.PageNode) {

	if !permissions.HasObjectPermission(r, p, "pages:edit") {
		auth.Fail(
			http.StatusForbidden,
			"User does not have permission to edit this page",
		)
		return
	}

	var instance, err = Specific(r.Context(), *p)
	except.Assert(
		err == nil, 500,
		err,
	)

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

		if !adminForm.HasChanged() {
			logger.Warnf("No changes detected for page: %s", page.Reference().Title)
			return nil
		}

		if page, ok := d.(SaveablePage); ok {
			err = UpdatePage(QuerySet(), ctx, page)
		} else {
			var n = d.(*models.PageNode)
			err = QuerySet().UpdateNode(ctx, n.Title, n.Path, n.Depth, n.Numchild, n.UrlPath, n.Slug, int64(n.StatusFlags), n.PageID, n.ContentType, n.PK)

		}
		if err != nil {
			return err
		}

		auditlogs.Log("pages:edit", logger.INF, p, map[string]interface{}{
			"page_id": page.ID(),
			"label":   page.Reference().Title,
		})

		return django.Task("[TRANSACTION] Fixing tree structure upon manual page node save", func(app *django.Application) error {
			return FixTree(pageApp.QuerySet(), ctx)
		})
	}

	var view = &views.FormView[*admin.AdminModelForm[modelforms.ModelForm[attrs.Definer]]]{
		BaseView: views.BaseView{
			AllowedMethods:  []string{http.MethodGet, http.MethodPost},
			BaseTemplateKey: admin.BASE_KEY,
			TemplateName:    "pages/admin/edit_page.tmpl",
			GetContextFn: func(req *http.Request) (ctx.Context, error) {
				var context = admin.NewContext(req, admin.AdminSite, nil)

				context.Set("app", a)
				context.Set("model", m)
				context.Set("page_object", page)
				var backURL string
				if q := req.URL.Query().Get("next"); q != "" {
					backURL = q
				}
				context.Set("BackURL", backURL)
				context.Set("PostURL", django.Reverse("admin:pages:edit", p.Reference().ID()))

				var breadcrumbs, err = getPageBreadcrumbs(r, p, false)
				if err != nil {
					return nil, err
				}

				context.SetPage(admin.PageOptions{
					TitleFn:     fields.S("Edit %q", page.Reference().Title),
					BreadCrumbs: breadcrumbs,
				})

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
			var listViewURL = django.Reverse("admin:pages:list", instance.(Page).Reference().ID())
			http.Redirect(w, r, listViewURL, http.StatusSeeOther)
		},
	}

	if err := views.Invoke(view, w, r); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
