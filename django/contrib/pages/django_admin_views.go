package pages

import (
	"net/http"

	"github.com/Nigel2392/django"
	"github.com/Nigel2392/django/contrib/admin"
	"github.com/Nigel2392/django/contrib/pages/models"
	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/core/ctx"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/django/views"
	"github.com/Nigel2392/django/views/list"
	"github.com/Nigel2392/mux"
	"github.com/a-h/templ"
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
	var columns = make([]list.ListColumn[attrs.Definer], 0, len(m.ListView.Fields)+1)
	for _, field := range m.ListView.Fields {
		columns = append(
			columns,
			m.GetColumn(m.ListView, field),
		)
	}

	columns = append(columns, &listButtonColumn[attrs.Definer]{
		buttonText: func(defs attrs.Definitions, row attrs.Definer) string {
			var p = row.(*models.PageNode)
			if p.Numchild > 0 {
				return fields.T("View Children")
			}
			return fields.T("Edit Page")
		},
		getURL: func(defs attrs.Definitions, row attrs.Definer) string {
			var primaryField = defs.Primary()
			if primaryField == nil {
				return ""
			}

			if row.(*models.PageNode).Numchild > 0 {
				return django.Reverse(
					"admin:pages:list",
					primaryField.GetValue(),
				)
			} else {
				return django.Reverse(
					"admin:pages:edit",
					primaryField.GetValue(),
				)
			}
		},
		icon: templ.Raw("fa fa-eye"),
	})

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
			TemplateName:    "pages/admin/list.tmpl",
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
				return django.Reverse("admin:pages:edit", a.Name, m.Name, primaryField.GetValue())
			})
		},
	}

	views.Invoke(view, w, r)
}

func addPageHandler(w http.ResponseWriter, r *http.Request, a *admin.AppDefinition, m *admin.ModelDefinition, p *models.PageNode) {

}

func editPageHandler(w http.ResponseWriter, r *http.Request, a *admin.AppDefinition, m *admin.ModelDefinition, p *models.PageNode) {

}
