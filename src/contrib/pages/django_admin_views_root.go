package pages

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	autherrors "github.com/Nigel2392/go-django/src/contrib/auth/auth_errors"
	models "github.com/Nigel2392/go-django/src/contrib/pages/page_models"
	auditlogs "github.com/Nigel2392/go-django/src/contrib/reports/audit_logs"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
	"github.com/Nigel2392/go-django/src/permissions"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/go-django/src/views/list"
	"github.com/Nigel2392/mux"
)

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
					return trans.T("View Live")
				},
				URL: func(defs attrs.Definitions, row attrs.Definer) string {
					return URLPath(row.(*models.PageNode))
				},
			},
			{
				Show: func(defs attrs.Definitions, row attrs.Definer) bool {
					// return row.(*models.PageNode).Numchild > 0
					return permissions.HasObjectPermission(r, row, "pages:add")
				},
				Text: func(defs attrs.Definitions, row attrs.Definer) string {
					return trans.T("Add Child")
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
					return trans.T("Edit Page")
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
					return django.AppInstalled("auditlogs") && permissions.HasObjectPermission(r, row, "auditlogs:list")
				},
				Text: func(defs attrs.Definitions, row attrs.Definer) string {
					return trans.T("History")
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
					TitleFn:    trans.S("Root Pages"),
					SubtitleFn: trans.S("View all root pages"),
				})

				return context, nil
			},
		},
		GetListFn: func(amount, offset uint) ([]attrs.Definer, error) {
			var ctx = r.Context()
			var nodes, err = QuerySet().GetNodesByDepth(ctx, 0, models.StatusFlagNone, int32(offset), int32(amount))
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

	var definitions = ListRootDefinitions()
	definitions = FilterCreatableDefinitions(
		definitions,
	)
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
				TitleFn:    trans.S("Choose Page Type"),
				SubtitleFn: trans.S("Select the type of page you want to create"),
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
				ctx, ref.Title, ref.Path, ref.Depth, ref.Numchild, ref.UrlPath, ref.Slug, int64(ref.StatusFlags), ref.PageID, ref.ContentType, ref.LatestRevisionID, ref.PK,
			)
		} else if n, ok := d.(*models.PageNode); ok {
			_, err = qs.InsertNode(
				ctx, n.Title, n.Path, n.Depth, n.Numchild, n.UrlPath, n.Slug, int64(n.StatusFlags), n.PageID, n.ContentType, n.LatestRevisionID,
			)
		} else {
			err = fmt.Errorf("invalid page type: %T", d)
		}
		if err != nil {
			return err
		}

		var addData = map[string]interface{}{
			"cType":  cType.TypeName(),
			"pageId": ref.PageID,
		}

		auditlogs.Log(
			"pages:add",
			logger.INF,
			page.Reference(),
			addData,
		)

		return nil
		//return django.Task("[TRANSACTION] Fixing tree structure upon manual page node save", func(app *django.Application) error {
		//	return FixTree(pageApp.QuerySet(), ctx)
		//})
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
					TitleFn: trans.S("Add %q", cType.Model()),
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
