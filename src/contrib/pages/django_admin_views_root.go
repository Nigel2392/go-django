package pages

import (
	"context"
	"fmt"
	"net/http"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/contrib/messages"
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
		admin.ReLogin(w, r, r.URL.Path)
		return
	}

	var next = django.Reverse(
		"admin:pages",
	)

	var columns = make([]list.ListColumn[attrs.Definer], len(m.ListView.Fields)+1)
	for i, field := range m.ListView.Fields {
		columns[i+1] = m.GetColumn(r.Context(), m.ListView, field)
	}

	columns[0] = columns[1]
	columns[1] = &admin.ListActionsColumn[attrs.Definer]{
		Actions: getListActions(next),
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
			var qs = NewPageQuerySet().WithContext(r.Context())
			var nodes, err = qs.GetNodesByDepth(0, StatusFlagNone, int32(offset), int32(amount))
			if err != nil {
				return nil, err
			}
			var items = make([]attrs.Definer, 0, len(nodes))
			for _, n := range nodes {
				n := n
				items = append(items, n)
			}
			return items, nil
		},
		TitleFieldColumn: func(lc list.ListColumn[attrs.Definer]) list.ListColumn[attrs.Definer] {
			return list.TitleFieldColumn(lc, func(r *http.Request, defs attrs.Definitions, instance attrs.Definer) string {
				if !permissions.HasObjectPermission(r, instance, "pages:edit") {
					return ""
				}

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
		admin.ReLogin(w, r, r.URL.Path)
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
		admin.ReLogin(w, r, r.URL.Path)
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
		cType      = cTypeDef.ContentType()
		page       = attrs.NewObject[pageDefiner](cType)
		fieldDefs  = page.FieldDefs()
		definition = DefinitionForObject(page)
		panels     []admin.Panel
	)
	if definition == nil || definition.AddPanels == nil {
		panels = make([]admin.Panel, 0, fieldDefs.Len())
		for _, def := range fieldDefs.Fields() {
			var formField = def.FormField()
			if formField == nil {
				continue
			}

			panels = append(panels, admin.FieldPanel(def.Name()))
		}
	} else {
		panels = definition.AddPanels(
			r, page,
		)
	}

	var form = modelforms.NewBaseModelForm[attrs.Definer](r.Context(), page)
	form.WithContext(r.Context())
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
			if !ref.StatusFlags.Is(StatusFlagPublished) {
				ref.StatusFlags |= StatusFlagPublished
			}
		}

		switch page := d.(type) {
		case *PageNode:
			ref = page
		case Page:
			ref.PageObject = page
		default:
			return fmt.Errorf("invalid page type: %T", d)
		}

		err = NewPageQuerySet().
			WithContext(ctx).
			AddRoot(ref)
		if err != nil {
			return err
		}

		var addData = map[string]interface{}{
			"cType":  cType.TypeName(),
			"pageId": ref.PageID,
		}

		auditlogs.Log(ctx,
			"pages:add",
			logger.INF,
			page.Reference(),
			addData,
		)

		if publishPage {
			auditlogs.Log(ctx,
				"pages:publish",
				logger.INF,
				page.Reference(),
				map[string]interface{}{
					"page_id": ref.PageID,
					"label":   ref.Title,
				},
			)
		}

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

			messages.Success(r, "Root page created successfully")

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
