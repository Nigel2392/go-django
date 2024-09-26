package pages

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"slices"
	"strconv"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	autherrors "github.com/Nigel2392/go-django/src/contrib/auth/auth_errors"
	"github.com/Nigel2392/go-django/src/contrib/pages/models"
	auditlogs "github.com/Nigel2392/go-django/src/contrib/reports/audit_logs"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
	"github.com/Nigel2392/go-django/src/permissions"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/go-django/src/views/list"
	"github.com/Nigel2392/mux"
)

type pageDefiner interface {
	attrs.Definer
	Page
}

func pageAdminAppHandler(fn func(http.ResponseWriter, *http.Request, *admin.AppDefinition, *admin.ModelDefinition)) mux.Handler {
	return mux.NewHandler(func(w http.ResponseWriter, req *http.Request) {
		var app, ok = admin.AdminSite.Apps.Get(AdminPagesAppName)
		if !ok {
			except.Fail(http.StatusBadRequest, "App not found")
			return
		}

		model, ok := app.Models.Get(AdminPagesModelPath)
		if !ok {
			except.Fail(http.StatusBadRequest, "Model not found")
			return
		}

		fn(w, req, app, model)
	})
}

func pageHandler(fn func(http.ResponseWriter, *http.Request, *admin.AppDefinition, *admin.ModelDefinition, *models.PageNode)) mux.Handler {
	return mux.NewHandler(func(w http.ResponseWriter, req *http.Request) {

		var (
			ctx       = req.Context()
			routeVars = mux.Vars(req)
			pageID    = routeVars.GetInt("page_id")
		)
		if pageID == 0 {
			except.Fail(http.StatusNotFound, "invalid page id")
			return
		}

		var page, err = QuerySet().GetNodeByID(ctx, int64(pageID))
		if err != nil {
			except.Fail(http.StatusNotFound, "Failed to get page")
			return
		}

		var handler = pageAdminAppHandler(func(w http.ResponseWriter, req *http.Request, app *admin.AppDefinition, model *admin.ModelDefinition) {
			fn(w, req, app, model, &page)
		})

		handler.ServeHTTP(w, req)
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
	} else {
		breadcrumbs = append(breadcrumbs, admin.BreadCrumb{
			Title: "Root Pages",
			URL:   django.Reverse("admin:pages"),
		})
	}
	return breadcrumbs, nil
}

func getPageActions(rq *http.Request, p *models.PageNode) []admin.Action {
	var actions = make([]admin.Action, 0)
	if p.ID() == 0 {
		return actions
	}

	if p.StatusFlags.Is(models.StatusFlagPublished) {
		actions = append(actions, admin.Action{
			Icon:   "icon-view",
			Target: "_blank",
			Title:  fields.T("View Live"),
			URL: path.Join(
				pageApp.routePrefix, p.UrlPath,
			),
		})
	}

	if django.AppInstalled("auditlogs") &&
		permissions.HasPermission(rq, "auditlogs:list") {
		var u = django.Reverse(
			"admin:auditlogs",
		)

		var url, err = url.Parse(u)
		if err != nil {
			return actions
		}

		var q = url.Query()
		q.Set(
			"object_id",
			strconv.Itoa(int(p.ID())),
		)
		q.Set(
			"content_type",
			contenttypes.NewContentType(p).ShortTypeName(),
		)
		url.RawQuery = q.Encode()

		actions = append(actions, admin.Action{
			Icon:  "icon-history",
			Title: fields.T("History"),
			URL:   url.String(),
		})
	}

	return actions
}

func addNextUrl(current string, next string) string {
	if next == "" {
		return current
	}
	if current == "" {
		assert.Fail("current url is empty")
		return ""
	}

	var u, err = url.Parse(current)
	if err != nil {
		return current
	}
	var q = u.Query()
	q.Set("next", next)
	u.RawQuery = q.Encode()
	return u.String()
}

func listPageHandler(w http.ResponseWriter, r *http.Request, a *admin.AppDefinition, m *admin.ModelDefinition, p *models.PageNode) {

	if !permissions.HasObjectPermission(r, p, "pages:list") {
		autherrors.Fail(
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
				Show: func(defs attrs.Definitions, row attrs.Definer) bool { return true },
				Text: func(defs attrs.Definitions, row attrs.Definer) string {
					return fields.T("View Live")
				},
				URL: func(defs attrs.Definitions, row attrs.Definer) string {
					return path.Join(pageApp.routePrefix, row.(*models.PageNode).UrlPath)
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
					return addNextUrl(
						u, next,
					)
				},
			},
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
					return addNextUrl(
						u, next,
					)
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
			{
				Show: func(defs attrs.Definitions, row attrs.Definer) bool {
					return django.AppInstalled("auditlogs") && permissions.HasObjectPermission(r, row, "auditlogs:list")
				},
				Text: func(defs attrs.Definitions, row attrs.Definer) string {
					return fields.T("History")
				},
				URL: func(defs attrs.Definitions, row attrs.Definer) string {
					var u = django.Reverse(
						"admin:auditlogs",
					)
					var url, err = url.Parse(u)
					if err != nil {
						return u
					}
					var q = url.Query()
					q.Set("object_id", strconv.Itoa(int(row.(*models.PageNode).ID())))
					q.Set("content_type", contenttypes.NewContentType(row).ShortTypeName())
					url.RawQuery = q.Encode()
					return url.String()
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
			except.Fail(http.StatusInternalServerError, err)
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
					Actions:     getPageActions(r, p),
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
				return addNextUrl(
					u, next,
				)
			})
		},
	}

	views.Invoke(view, w, r)
}

func listRootPageHandler(w http.ResponseWriter, r *http.Request, a *admin.AppDefinition, m *admin.ModelDefinition) {
	if !permissions.HasPermission(r, "pages:list") {
		autherrors.Fail(
			http.StatusForbidden,
			"User does not have permission to view this page",
		)
		return
	}

	var next = django.Reverse(
		"admin:pages",
	)

	var columns = make([]list.ListColumn[attrs.Definer], len(m.ListView.Fields)+1)
	for i, field := range m.ListView.Fields {
		columns[i+1] = m.GetColumn(m.ListView, field)
	}

	columns[0] = columns[1]
	columns[1] = &admin.ListActionsColumn[attrs.Definer]{
		Actions: []*admin.ListAction[attrs.Definer]{
			{
				Show: func(defs attrs.Definitions, row attrs.Definer) bool { return true },
				Text: func(defs attrs.Definitions, row attrs.Definer) string {
					return fields.T("View Live")
				},
				URL: func(defs attrs.Definitions, row attrs.Definer) string {
					return path.Join(pageApp.routePrefix, row.(*models.PageNode).UrlPath)
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
					return addNextUrl(
						u, next,
					)
				},
			},
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
					return addNextUrl(
						u, next,
					)
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
					var u = django.Reverse(
						"admin:pages:list",
						primaryField.GetValue(),
					)
					return addNextUrl(
						u, next,
					)
				},
			},
			{
				Show: func(defs attrs.Definitions, row attrs.Definer) bool {
					return django.AppInstalled("auditlogs") && permissions.HasObjectPermission(r, row, "auditlogs:list")
				},
				Text: func(defs attrs.Definitions, row attrs.Definer) string {
					return fields.T("History")
				},
				URL: func(defs attrs.Definitions, row attrs.Definer) string {
					var u = django.Reverse(
						"admin:auditlogs",
					)
					var url, err = url.Parse(u)
					if err != nil {
						return u
					}
					var q = url.Query()
					q.Set("object_id", strconv.Itoa(int(row.(*models.PageNode).ID())))
					q.Set("content_type", contenttypes.NewContentType(row).ShortTypeName())
					url.RawQuery = q.Encode()
					return url.String()
				},
			},
		},
	}

	var amount = m.ListView.PerPage
	if amount == 0 {
		amount = 25
	}

	var view = &list.View[attrs.Definer]{
		ListColumns:   columns,
		DefaultAmount: amount,
		BaseView: views.BaseView{
			AllowedMethods:  []string{http.MethodGet, http.MethodPost},
			BaseTemplateKey: admin.BASE_KEY,
			TemplateName:    "pages/admin/admin_list_root.tmpl",
			GetContextFn: func(req *http.Request) (ctx.Context, error) {
				var context = admin.NewContext(
					req, admin.AdminSite, nil,
				)

				context.Set("app", a)
				context.Set("model", m)
				context.SetPage(admin.PageOptions{
					TitleFn:    fields.S("Root Pages"),
					SubtitleFn: fields.S("View all root pages"),
				})

				return context, nil
			},
		},
		GetListFn: func(amount, offset uint, include []string) ([]attrs.Definer, error) {
			var ctx = r.Context()
			var nodes, err = QuerySet().GetNodesByDepth(ctx, 0, int32(amount), int32(offset))
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

				return django.Reverse(
					"admin:pages:edit",
					primaryField.GetValue(),
				)
			})
		},
	}

	views.Invoke(view, w, r)
}

func chooseRootPageTypeHandler(w http.ResponseWriter, r *http.Request, a *admin.AppDefinition, m *admin.ModelDefinition) {

	if !permissions.HasPermission(r, "pages:add") {
		autherrors.Fail(
			http.StatusForbidden,
			"User does not have permission to add a page",
		)
		return
	}

	var definitions = ListDefinitions()
	var view = &views.BaseView{
		AllowedMethods:  []string{http.MethodGet},
		BaseTemplateKey: admin.BASE_KEY,
		TemplateName:    "pages/admin/choose_root_page_type.tmpl",
		GetContextFn: func(req *http.Request) (ctx.Context, error) {
			var context = admin.NewContext(req, admin.AdminSite, nil)

			context.Set("app", a)
			context.Set("model", m)
			context.Set("definitions", definitions)

			var next = req.URL.Query().Get("next")
			if next != "" {
				context.Set("BackURL", next)
			}

			context.SetPage(admin.PageOptions{
				TitleFn:    fields.S("Choose Page Type"),
				SubtitleFn: fields.S("Select the type of page you want to create"),
			})

			return context, nil
		},
	}

	views.Invoke(view, w, r)
}

func addRootPageHandler(w http.ResponseWriter, r *http.Request, a *admin.AppDefinition, m *admin.ModelDefinition) {
	if !permissions.HasPermission(r, "pages:add") {
		autherrors.Fail(
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
		except.Fail(http.StatusBadRequest, "app_label and model_name are required")
		return
	}

	var cTypeDef = contenttypes.DefinitionForPackage(app_label, model_name)
	if cTypeDef == nil {
		except.Fail(http.StatusNotFound, "content type not found")
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
			panels[i] = admin.FieldPanel(
				def.Name(),
			)
		}
	} else {
		panels = definition.AddPanels(
			r, page,
		)
	}

	var form = modelforms.NewBaseModelForm[attrs.Definer](page)
	var adminForm = admin.NewAdminModelForm[modelforms.ModelForm[attrs.Definer]](
		form, panels...,
	)

	adminForm.Load()

	form.SaveInstance = func(ctx context.Context, d attrs.Definer) (err error) {

		var publishPage = r.FormValue("publish-page") == "publish-page" && permissions.HasPermission(
			r, "pages:publish",
		)

		var ref = d.(Page).Reference()
		if publishPage {
			if !ref.StatusFlags.Is(models.StatusFlagPublished) {
				ref.StatusFlags |= models.StatusFlagPublished
			}
		}

		var qs = QuerySet()
		err = CreateRootNode(qs, ctx, ref)
		if err != nil {
			return err
		}

		if page, ok := d.(SaveablePage); ok {
			err = SavePage(
				QuerySet(), ctx, nil, page,
			)
			if err != nil {
				return err
			}

			ref.PageID = page.ID()
			err = qs.UpdateNode(
				ctx, ref.Title, ref.Path, ref.Depth, ref.Numchild, ref.UrlPath, ref.Slug, int64(ref.StatusFlags), ref.PageID, ref.ContentType, ref.PK,
			)
		} else if n, ok := d.(*models.PageNode); ok {
			_, err = qs.InsertNode(
				ctx, n.Title, n.Path, n.Depth, n.Numchild, n.UrlPath, n.Slug, int64(n.StatusFlags), n.PageID, n.ContentType,
			)
		} else {
			err = fmt.Errorf("invalid page type: %T", d)
		}
		if err != nil {
			return err
		}

		var addData = map[string]interface{}{
			"cType": cType.PkgPath(),
		}

		auditlogs.Log(
			"pages:add",
			logger.INF,
			page.Reference(),
			addData,
		)

		return django.Task("[TRANSACTION] Fixing tree structure upon manual page node save", func(app *django.Application) error {
			return FixTree(pageApp.QuerySet(), ctx)
		})
	}

	var view = &views.FormView[*admin.AdminModelForm[modelforms.ModelForm[attrs.Definer]]]{
		BaseView: views.BaseView{
			AllowedMethods:  []string{http.MethodGet, http.MethodPost},
			BaseTemplateKey: admin.BASE_KEY,
			TemplateName:    "pages/admin/add_root_page.tmpl",
			GetContextFn: func(req *http.Request) (ctx.Context, error) {
				var context = admin.NewContext(req, admin.AdminSite, nil)

				context.Set("app", a)
				context.Set("model", m)

				var backURL string
				if q := req.URL.Query().Get("next"); q != "" {
					backURL = q
				}
				context.Set("BackURL", backURL)
				context.Set("PostURL", django.Reverse(
					"admin:pages:root_add",
					cType.AppLabel(),
					cType.Model(),
				))

				context.SetPage(admin.PageOptions{
					TitleFn: fields.S("Add %q", cType.Model()),
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
		except.Fail(500, err)
		return
	}
}

func choosePageTypeHandler(w http.ResponseWriter, r *http.Request, a *admin.AppDefinition, m *admin.ModelDefinition, p *models.PageNode) {

	if !permissions.HasObjectPermission(r, p, "pages:add") {
		autherrors.Fail(
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
				Actions:     getPageActions(r, p),
			})

			return context, nil
		},
	}

	views.Invoke(view, w, r)
}

func addPageHandler(w http.ResponseWriter, r *http.Request, a *admin.AppDefinition, m *admin.ModelDefinition, p *models.PageNode) {
	if !permissions.HasObjectPermission(r, p, "pages:add") {
		autherrors.Fail(
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
		except.Fail(http.StatusBadRequest, "app_label and model_name are required")
		return
	}

	var cTypeDef = contenttypes.DefinitionForPackage(app_label, model_name)
	if cTypeDef == nil {
		except.Fail(http.StatusNotFound, "content type not found")
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
			panels[i] = admin.FieldPanel(
				def.Name(),
			)
		}
	} else {
		panels = definition.AddPanels(
			r, page,
		)
	}

	var form = modelforms.NewBaseModelForm[attrs.Definer](page)
	var adminForm = admin.NewAdminModelForm[modelforms.ModelForm[attrs.Definer]](
		form, panels...,
	)

	adminForm.Load()

	form.SaveInstance = func(ctx context.Context, d attrs.Definer) (err error) {

		var publishPage = r.FormValue("publish-page") == "publish-page" && permissions.HasObjectPermission(
			r, p, "pages:publish",
		)

		var ref = d.(Page).Reference()
		if publishPage {
			if !ref.StatusFlags.Is(models.StatusFlagPublished) {
				ref.StatusFlags |= models.StatusFlagPublished
			}
		}

		if page, ok := d.(SaveablePage); ok {
			err = SavePage(
				QuerySet(), ctx, p, page,
			)
		} else {
			var n = d.(*models.PageNode)
			_, err = QuerySet().InsertNode(
				ctx, n.Title, n.Path, n.Depth, n.Numchild, n.UrlPath, n.Slug, int64(n.StatusFlags), n.PageID, n.ContentType,
			)
		}
		if err != nil {
			return err
		}

		var addData = map[string]interface{}{
			"cType": cType.PkgPath(),
		}

		if p != nil && p.ID() > 0 {
			addData["parent"] = p.ID()
		}

		auditlogs.Log(
			"pages:add",
			logger.INF,
			page.Reference(),
			addData,
		)

		if p != nil && p.ID() > 0 {
			auditlogs.Log("pages:add_child", logger.INF, p, map[string]interface{}{
				"page_id": ref.ID(),
				"label":   ref.Title,
				"cType":   cType.PkgPath(),
			})
		}

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
					Actions:     getPageActions(r, p),
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
		except.Fail(500, err)
		return
	}
}

func editPageHandler(w http.ResponseWriter, r *http.Request, a *admin.AppDefinition, m *admin.ModelDefinition, p *models.PageNode) {

	if !permissions.HasObjectPermission(r, p, "pages:edit") {
		autherrors.Fail(
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

	if err := r.ParseForm(); err != nil {
		except.Fail(500, err)
		return
	}

	var publishPage = r.FormValue("publish-page") == "publish-page" && permissions.HasObjectPermission(
		r, p, "pages:publish",
	)

	var unpublishPage = r.FormValue("unpublish-page") == "unpublish-page" && permissions.HasObjectPermission(
		r, p, "pages:publish",
	)

	form.SaveInstance = func(ctx context.Context, d attrs.Definer) error {

		if !adminForm.HasChanged() && !publishPage && !unpublishPage {
			logger.Warnf("No changes detected for page: %s", page.Reference().Title)
			return nil
		}

		if page, ok := d.(SaveablePage); ok {

			var ref = page.Reference()
			if publishPage && !ref.StatusFlags.Is(models.StatusFlagPublished) {
				ref.StatusFlags |= models.StatusFlagPublished
			}

			if ref.Numchild == 0 && unpublishPage && ref.StatusFlags.Is(models.StatusFlagPublished) {
				ref.StatusFlags &^= models.StatusFlagPublished
			}

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
				context.Set("is_published", p.StatusFlags.Is(models.StatusFlagPublished))
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
					Actions:     getPageActions(r, p),
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
			var page = instance.(Page)
			var ref = page.Reference()
			var redirectURL string
			if unpublishPage && ref.Numchild > 0 && ref.StatusFlags.Is(models.StatusFlagPublished) {
				redirectURL = django.Reverse("admin:pages:unpublish", ref.ID())
			} else {
				redirectURL = django.Reverse("admin:pages:list", ref.ID())
			}
			http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		},
	}

	if err := views.Invoke(view, w, r); err != nil {
		except.Fail(500, err)
		return
	}
}

func deletePageHandler(w http.ResponseWriter, r *http.Request, a *admin.AppDefinition, m *admin.ModelDefinition, p *models.PageNode) {
	if !permissions.HasObjectPermission(r, p, "pages:delete") {
		autherrors.Fail(
			http.StatusForbidden,
			"User does not have permission to delete this page",
		)
		return
	}

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			except.Fail(500, "Failed to parse form")
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

		var parent models.PageNode
		if p.Depth > 0 {
			parent, err = ParentNode(
				QuerySet(),
				r.Context(),
				p.Path,
				int(p.Depth),
			)
			if err != nil {
				except.Fail(http.StatusInternalServerError, err)
				return
			}
		}

		if page, ok := page.(DeletablePage); ok {
			if err := DeletePage(QuerySet(), r.Context(), page); err != nil {
				except.Fail(500, "Failed to delete page: %s", err)
				return
			}
		} else {
			if err := DeleteNode(QuerySet(), r.Context(), p.ID(), p.Path, p.Depth); err != nil {
				except.Fail(500, "Failed to delete page: %s", err)
				return
			}

			logger.Warnf("Page %q (%v) deleted but has no Delete() method", p.Title, p.ID())
		}

		auditlogs.Log("pages:delete", logger.WRN, p, map[string]interface{}{
			"page_id": p.ID(),
			"label":   p.Title,
		})

		if p.Depth > 0 {
			http.Redirect(w, r, django.Reverse("admin:pages:list", parent.ID()), http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, django.Reverse("admin:pages:root_list"), http.StatusSeeOther)
		return
	}

	var view = &views.BaseView{
		AllowedMethods:  []string{http.MethodGet},
		BaseTemplateKey: admin.BASE_KEY,
		TemplateName:    "pages/admin/delete_page.tmpl",
		GetContextFn: func(req *http.Request) (ctx.Context, error) {
			var context = admin.NewContext(req, admin.AdminSite, nil)

			context.Set("app", a)
			context.Set("model", m)
			context.Set("page_object", p)

			var breadcrumbs, err = getPageBreadcrumbs(r, p, false)
			if err != nil {
				return nil, err
			}

			context.SetPage(admin.PageOptions{
				TitleFn:     fields.S("Delete %q", p.Title),
				SubtitleFn:  fields.S("Are you sure you want to delete this page?"),
				BreadCrumbs: breadcrumbs,
				Actions:     getPageActions(r, p),
			})

			return context, nil
		},
	}

	if err := views.Invoke(view, w, r); err != nil {
		except.Fail(500, err)
		return
	}
}

func unpublishPageHandler(w http.ResponseWriter, r *http.Request, a *admin.AppDefinition, m *admin.ModelDefinition, p *models.PageNode) {
	if !permissions.HasObjectPermission(r, p, "pages:publish") {
		autherrors.Fail(
			http.StatusForbidden,
			"User does not have permission to unpublish this page",
		)
		return
	}

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			except.Fail(500, "Failed to parse form")
			return
		}

		var unpublishChildren = r.FormValue("unpublish-children") == "unpublish-children"
		if err := UnpublishNode(QuerySet(), r.Context(), p, unpublishChildren); err != nil {
			except.Fail(500, "Failed to unpublish page: %s", err)
			return
		}

		auditlogs.Log("pages:unpublish", logger.WRN, p, map[string]interface{}{
			"unpublish_children": unpublishChildren,
			"page_id":            p.ID(),
			"label":              p.Title,
		})

		http.Redirect(w, r, django.Reverse("admin:pages:list", p.ID()), http.StatusSeeOther)
		return
	}

	var view = &views.BaseView{
		AllowedMethods:  []string{http.MethodGet},
		BaseTemplateKey: admin.BASE_KEY,
		TemplateName:    "pages/admin/unpublish_page.tmpl",
		GetContextFn: func(req *http.Request) (ctx.Context, error) {
			var context = admin.NewContext(req, admin.AdminSite, nil)

			context.Set("app", a)
			context.Set("model", m)
			context.Set("page_object", p)

			var breadcrumbs, err = getPageBreadcrumbs(r, p, false)
			if err != nil {
				return nil, err
			}

			context.SetPage(admin.PageOptions{
				TitleFn:     fields.S("Unpublish %q", p.Title),
				SubtitleFn:  fields.S("Unpublishing a page will remove it from the live site\nOptionally, you can unpublish all child pages"),
				BreadCrumbs: breadcrumbs,
				Actions:     getPageActions(r, p),
			})

			return context, nil
		},
	}

	if err := views.Invoke(view, w, r); err != nil {
		except.Fail(500, err)
		return
	}
}
