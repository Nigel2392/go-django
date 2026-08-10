package generic

import (
	"context"
	"net/http"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/admin/components"
	"github.com/Nigel2392/go-django/contrib/admin/components/columns"
	"github.com/Nigel2392/go-django/contrib/admin/components/menu"
	"github.com/Nigel2392/go-django/contrib/filters"
	"github.com/Nigel2392/go-django/contrib/shop/internal/app"
	"github.com/Nigel2392/go-django/contrib/shop/models"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/permissions"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/go-django/src/views/list"
)

type ListViewConfig[T attrs.Definer] struct {
	Model              T
	PerPage            int
	UsePanelFilters    bool
	ListTitle          func(context.Context) string
	ListSubtitle       func(context.Context) string
	Filters            []filters.FilterSpec[T]
	GetEditLink        func(r *http.Request, v T) string
	GetColumns         func(r *http.Request) ([]list.ListColumn[T], error)
	GetListActions     func(r *http.Request) ([]*columns.ListAction[T], error)
	GetQuerySet        func(r *http.Request) *queries.QuerySet[T]
	GetPageBreadCrumbs func(r *http.Request) []admin.BreadCrumb
	GetPageActions     func(r *http.Request) []admin.Action
	GetHeaderActions   func(r *http.Request) []components.ShowableComponent
	GetPageSidePanels  func(r *http.Request) *menu.SidePanels
}

func (l ListViewConfig[T]) getPageBreadCrumbs(r *http.Request) []admin.BreadCrumb {
	if l.GetPageBreadCrumbs != nil {
		return l.GetPageBreadCrumbs(r)
	}
	return []admin.BreadCrumb{}
}

func (l ListViewConfig[T]) getPageActions(r *http.Request) []admin.Action {
	if l.GetPageActions != nil {
		return l.GetPageActions(r)
	}
	return []admin.Action{}
}

func (l ListViewConfig[T]) getHeaderActions(r *http.Request) []components.ShowableComponent {
	if l.GetHeaderActions != nil {
		return l.GetHeaderActions(r)
	}
	return []components.ShowableComponent{}
}

func (l ListViewConfig[T]) getPageSidePanels(r *http.Request) *menu.SidePanels {
	if l.GetPageSidePanels != nil {
		return l.GetPageSidePanels(r)
	}
	return nil
}

func (l ListViewConfig[T]) ServeHTTP(w http.ResponseWriter, r *http.Request, shop *app.ShopAppConfig) {
	if !permissions.HasObjectPermission(r, &models.Product{}, "products:list") {
		admin.ReLogin(w, r, r.URL.Path)
		return
	}

	var _columns, err = l.GetColumns(r)
	if err != nil {
		except.Fail(
			http.StatusInternalServerError,
			"could not retrieve columns: %v", err,
		)
	}

	var sortBuilder = columns.NewSortableColumnBuilder(l.Model)
	for i := range _columns {
		_columns[i] = sortBuilder.AddColumn(_columns[i])
	}

	var cols []list.ListColumn[T]
	if l.GetListActions == nil {
		cols = _columns
	} else {
		cols = make([]list.ListColumn[T], 0, len(_columns)+1)

		copy(cols[1:], _columns)

		listActions, err := l.GetListActions(r)
		if err != nil {
			except.Fail(
				http.StatusInternalServerError,
				"could not retrieve columns: %v", err,
			)
		}

		cols[0] = cols[1]
		cols[1] = &columns.ListActionsColumn[T]{
			Actions: listActions,
		}
	}

	if l.GetEditLink != nil {
		cols[0] = list.TitleFieldColumn(
			cols[0], func(r *http.Request, defs attrs.Definitions, row T) string {
				return l.GetEditLink(r, row)
			},
		)
	}

	var amount = l.PerPage
	if amount == 0 {
		amount = 25
	}

	var qs *queries.QuerySet[T]
	if l.GetQuerySet != nil {
		qs = l.GetQuerySet(r)
	}

	if qs == nil {
		qs = queries.GetQuerySet(l.Model)
	}

	qs = qs.WithContext(r.Context())
	qs = sortBuilder.Sort(qs, r.URL.Query()["sort"])

	var filterForm *filters.Filters[T]
	if len(l.Filters) > 0 {
		filterForm = filters.NewFilters[T](r.Context(), "filters")
		for _, f := range l.Filters {
			filterForm.Add(f)
		}

		var err error
		qs, err = filterForm.Filter(r, r.URL.Query(), qs)
		if err != nil {
			except.AssertNil(err, http.StatusInternalServerError, err)
			return
		}
	}

	var view = &list.View[T]{
		ListColumns:     cols,
		DefaultAmount:   int(amount),
		Model:           l.Model,
		AllowedMethods:  []string{http.MethodGet, http.MethodPost},
		BaseTemplateKey: "shop",
		TemplateName:    "shop/admin/models/list.tmpl",
		ChangeContextFn: func(req *http.Request, qs *queries.QuerySet[T], viewCtx ctx.Context) (ctx.Context, error) {
			var context = admin.NewContext(
				req, admin.AdminSite, viewCtx,
			)

			var panel = l.getPageSidePanels(r)
			if l.UsePanelFilters && len(l.Filters) > 0 {
				if panel == nil {
					panel = &menu.SidePanels{
						Panels: []menu.SidePanel{},
					}
				}

				panel.Panels = append(panel.Panels,
					admin.SidePanelFilters(
						r, filterForm, list.PageFromContext[T](req.Context()),
					),
				)
			}

			if filterForm != nil {
				context.Set("filter", filterForm)
			}

			context.SetPage(admin.PageOptions{
				TitleFn:       l.ListTitle,
				SubtitleFn:    l.ListSubtitle,
				BreadCrumbs:   l.getPageBreadCrumbs(r),
				Actions:       l.getPageActions(r),
				HeaderActions: l.getHeaderActions(r),
				SidePanels:    panel,
			})

			return context, nil
		},

		QuerySet: func(r *http.Request) *queries.QuerySet[T] {

			return qs
		},
	}

	views.Invoke(view, w, r)
}
