package pages

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

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

func getListActions(next string) []*admin.ListAction[attrs.Definer] {
	return []*admin.ListAction[attrs.Definer]{
		{
			Show: func(r *http.Request, defs attrs.Definitions, row attrs.Definer) bool { return true },
			Text: func(r *http.Request, defs attrs.Definitions, row attrs.Definer) string {
				return trans.T(r.Context(), "View Live")
			},
			URL: func(r *http.Request, defs attrs.Definitions, row attrs.Definer) string {
				return URLPath(row.(*PageNode))
			},
		},
		{
			Show: func(r *http.Request, defs attrs.Definitions, row attrs.Definer) bool {
				// return row.(*PageNode).Numchild > 0
				return permissions.HasObjectPermission(r, row, "pages:add")
			},
			Text: func(r *http.Request, defs attrs.Definitions, row attrs.Definer) string {
				return trans.T(r.Context(), "Add Child")
			},
			URL: func(r *http.Request, defs attrs.Definitions, row attrs.Definer) string {
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
			Show: func(r *http.Request, defs attrs.Definitions, row attrs.Definer) bool {
				return permissions.HasObjectPermission(r, row, "pages:edit")
			},
			Text: func(r *http.Request, defs attrs.Definitions, row attrs.Definer) string {
				return trans.T(r.Context(), "Edit Page")
			},
			URL: func(r *http.Request, defs attrs.Definitions, row attrs.Definer) string {
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
			Show: func(r *http.Request, defs attrs.Definitions, row attrs.Definer) bool {
				return django.AppInstalled("auditlogs") && permissions.HasObjectPermission(r, row, "auditlogs:list")
			},
			Text: func(r *http.Request, defs attrs.Definitions, row attrs.Definer) string {
				return trans.T(r.Context(), "History")
			},
			URL: func(r *http.Request, defs attrs.Definitions, row attrs.Definer) string {
				var u = django.Reverse(
					"admin:auditlogs",
				)
				var url, err = url.Parse(u)
				if err != nil {
					return u
				}
				var q = url.Query()
				q.Set("filters-object_id", strconv.Itoa(int(row.(*PageNode).ID())))
				q.Set("filters-content_type", contenttypes.NewContentType(row).ShortTypeName())
				url.RawQuery = q.Encode()
				return url.String()
			},
		},
	}
}

func listPageHandler(w http.ResponseWriter, r *http.Request, a *admin.AppDefinition, m *admin.ModelDefinition, p *PageNode) {

	if !permissions.HasObjectPermission(r, p, "pages:list") {
		admin.ReLogin(w, r, r.URL.Path)
		return
	}

	var columns = make([]list.ListColumn[attrs.Definer], len(m.ListView.Fields)+1)
	for i, field := range m.ListView.Fields {
		columns[i+1] = m.GetColumn(r.Context(), m.ListView, field)
	}

	var next = django.Reverse(
		"admin:pages:list",
		p.ID(),
	)

	columns[0] = columns[1]
	columns[1] = &admin.ListActionsColumn[attrs.Definer]{
		Actions: getListActions(next),
	}

	var amount = m.ListView.PerPage
	if amount == 0 {
		amount = 25
	}

	var parent_object *PageNode
	var qs = NewPageQuerySet().WithContext(r.Context())
	if p.Depth > 0 {
		var parent, err = qs.ParentNode(
			p.Path,
			int(p.Depth),
		)
		if err != nil {
			except.Fail(http.StatusInternalServerError, err)
			return
		}
		parent_object = parent
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
					contentType.Label(r.Context()),
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
		GetListFn: func(amount, offset uint) ([]attrs.Definer, error) {
			var nodes, err = qs.GetChildNodes(p, StatusFlagNone, int32(offset), int32(amount))
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

func choosePageTypeHandler(w http.ResponseWriter, r *http.Request, a *admin.AppDefinition, m *admin.ModelDefinition, p *PageNode) {

	if !permissions.HasObjectPermission(r, p, "pages:add") {
		admin.ReLogin(w, r, r.URL.Path)
		return
	}

	var definitions []*PageDefinition
	if p.ContentType != "" && p.ContentType != contenttypes.NewContentType(p).TypeName() {
		definitions = ListDefinitionsForType(p.ContentType)
	} else {
		definitions = ListDefinitions()
	}
	definitions = FilterCreatableDefinitions(
		definitions,
	)
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
				TitleFn:     trans.S("Choose Page Type"),
				SubtitleFn:  trans.S("Select the type of page you want to create"),
				BreadCrumbs: breadcrumbs,
				Actions:     getPageActions(r, p),
			})

			return context, nil
		},
	}

	views.Invoke(view, w, r)
}

func addPageHandler(w http.ResponseWriter, r *http.Request, a *admin.AppDefinition, m *admin.ModelDefinition, p *PageNode) {
	if !permissions.HasObjectPermission(r, p, "pages:add") {
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

		var publishPage = r.FormValue("publish-page") == "publish-page" && permissions.HasObjectPermission(
			r, p, "pages:publish",
		)

		var ref = d.(Page).Reference()
		if publishPage {
			if !ref.StatusFlags.Is(StatusFlagPublished) {
				ref.StatusFlags |= StatusFlagPublished
			}
		}

		var qs = NewPageQuerySet().WithContext(ctx)
		switch page := d.(type) {
		case *PageNode:
			ref = page
		case Page:
			ref.PageObject = page
		default:
			return fmt.Errorf("invalid page type: %T", d)
		}
		err = qs.AddChildren(
			p, ref,
		)
		if err != nil {
			return err
		}

		var addData = map[string]interface{}{
			"cType": cType.PkgPath(),
		}

		if p != nil && p.ID() > 0 {
			addData["parent"] = p.ID()
		}

		auditlogs.Log(ctx,
			"pages:add",
			logger.INF,
			page.Reference(),
			addData,
		)

		if p != nil && p.ID() > 0 {
			auditlogs.Log(ctx, "pages:add_child", logger.INF, p, map[string]interface{}{
				"page_id": ref.ID(),
				"label":   ref.Title,
				"cType":   cType.PkgPath(),
			})
		}

		if publishPage {
			auditlogs.Log(ctx, "pages:publish", logger.INF, p, map[string]interface{}{
				"page_id": ref.ID(),
				"label":   ref.Title,
				"cType":   cType.PkgPath(),
			})
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
					TitleFn:     trans.S("Add %q", cType.Model()),
					BreadCrumbs: breadcrumbs,
					Actions:     getPageActions(r, p),
				})

				return context, nil
			},
		},
		GetFormFn: func(req *http.Request) *admin.AdminModelForm[modelforms.ModelForm[attrs.Definer]] {
			return adminForm
		},
		SuccessFn: func(w http.ResponseWriter, req *http.Request, form *admin.AdminModelForm[modelforms.ModelForm[attrs.Definer]]) {
			var instance = form.Instance()
			assert.False(instance == nil, "instance is nil after form submission")

			messages.Success(r, "Page created successfully")

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

func editPageHandler(w http.ResponseWriter, r *http.Request, a *admin.AppDefinition, m *admin.ModelDefinition, p *PageNode) {

	if !permissions.HasObjectPermission(r, p, "pages:edit") {
		admin.ReLogin(w, r, r.URL.Path)
		return
	}

	var instance, err = p.Specific(r.Context())
	except.Assert(
		err == nil, 500,
		err,
	)

	var page, ok = instance.(pageDefiner)
	if !ok {
		page = p
		// logger.Warnf("instance does not adhere to attrs.Definer: %T", instance)
	}

	var fieldDefs = page.FieldDefs()
	var definition = DefinitionForObject(page)
	var panels []admin.Panel
	if definition == nil || definition.EditPanels == nil {
		panels = make([]admin.Panel, 0, fieldDefs.Len())
		for _, def := range fieldDefs.Fields() {
			var formField = def.FormField()
			if formField == nil {
				continue
			}

			panels = append(panels, admin.FieldPanel(def.Name()))
		}
	} else {
		panels = definition.EditPanels(r, page)
	}

	var form = modelforms.NewBaseModelForm[attrs.Definer](r.Context(), page)
	form.WithContext(r.Context())
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

		var (
			wasPublished, wasUnpublished bool
		)

		var ref = page.Reference()
		if publishPage && !ref.StatusFlags.Is(StatusFlagPublished) {
			ref.StatusFlags |= StatusFlagPublished
			wasPublished = true
		}

		// If no children it is safe to unpublish the page straight away,
		// otherwise we will later redirect to an unpublish page- view.
		if ref.Numchild == 0 && unpublishPage && ref.StatusFlags.Is(StatusFlagPublished) {
			ref.StatusFlags &^= StatusFlagPublished
			wasUnpublished = true
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
			UpdateNode(ref)
		if err != nil {
			return err
		}

		auditlogs.Log(ctx, "pages:edit", logger.INF, p, map[string]interface{}{
			"page_id": page.ID(),
			"label":   page.Reference().Title,
		})

		if wasPublished {
			auditlogs.Log(ctx, "pages:publish", logger.INF, p, map[string]interface{}{
				"page_id": page.ID(),
				"label":   page.Reference().Title,
			})
		}

		if wasUnpublished {
			auditlogs.Log(ctx, "pages:unpublish", logger.INF, p, map[string]interface{}{
				"page_id": page.ID(),
				"label":   page.Reference().Title,
			})
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
			TemplateName:    "pages/admin/edit_page.tmpl",
			GetContextFn: func(req *http.Request) (ctx.Context, error) {
				var context = admin.NewContext(req, admin.AdminSite, nil)

				context.Set("app", a)
				context.Set("model", m)
				context.Set("page_object", page)
				context.Set("is_published", p.StatusFlags.Is(StatusFlagPublished))
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
					TitleFn:     trans.S("Edit %q", page.Reference().Title),
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

			messages.Success(r, "Page updated successfully")

			var redirectURL string
			if unpublishPage && ref.Numchild > 0 && ref.StatusFlags.Is(StatusFlagPublished) {
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

func deletePageHandler(w http.ResponseWriter, r *http.Request, a *admin.AppDefinition, m *admin.ModelDefinition, p *PageNode) {
	if !permissions.HasObjectPermission(r, p, "pages:delete") {
		admin.ReLogin(w, r, r.URL.Path)
		return
	}

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			except.Fail(500, "Failed to parse form")
			return
		}

		var err error
		var parent *PageNode
		var qs = NewPageQuerySet().WithContext(r.Context())
		if p.Depth > 0 {
			parent, err = qs.ParentNode(
				p.Path,
				int(p.Depth),
			)
			if err != nil {
				except.Fail(http.StatusInternalServerError, err)
				return
			}
		}

		if _, err := qs.Delete(p); err != nil {
			except.Fail(500, "Failed to delete page: %s", err)
			return
		}

		auditlogs.Log(r.Context(), "pages:delete", logger.WRN, p, map[string]interface{}{
			"page_id": p.ID(),
			"label":   p.Title,
		})

		messages.Warning(r, "Page deleted successfully")

		if p.Depth > 0 {
			http.Redirect(w, r, django.Reverse("admin:pages:list", parent.ID()), http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, django.Reverse("admin:pages"), http.StatusSeeOther)
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
				TitleFn:     trans.S("Delete %q", p.Title),
				SubtitleFn:  trans.S("Are you sure you want to delete this page?"),
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

func unpublishPageHandler(w http.ResponseWriter, r *http.Request, a *admin.AppDefinition, m *admin.ModelDefinition, p *PageNode) {
	if !permissions.HasObjectPermission(r, p, "pages:publish") {
		admin.ReLogin(w, r, r.URL.Path)
		return
	}

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			except.Fail(500, "Failed to parse form")
			return
		}

		var unpublishChildren = r.FormValue("unpublish-children") == "unpublish-children"
		var qs = NewPageQuerySet().WithContext(r.Context())
		if err := qs.UnpublishNode(p, unpublishChildren); err != nil {
			except.Fail(500, "Failed to unpublish page: %s", err)
			return
		}

		auditlogs.Log(r.Context(), "pages:unpublish", logger.WRN, p, map[string]interface{}{
			"unpublish_children": unpublishChildren,
			"page_id":            p.ID(),
			"label":              p.Title,
		})

		if unpublishChildren {
			messages.Warning(r, "Page and all child pages unpublished successfully")
		} else {
			messages.Warning(r, "Page unpublished successfully")
		}

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
				TitleFn:     trans.S("Unpublish %q", p.Title),
				SubtitleFn:  trans.S("Unpublishing a page will remove it from the live site\nOptionally, you can unpublish all child pages"),
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
