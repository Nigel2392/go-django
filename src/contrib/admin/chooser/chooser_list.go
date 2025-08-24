package chooser

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strconv"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/components"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/pagination"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/permissions"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/go-django/src/views/list"
)

var (
	_ views.View         = (*ChooserListPage[attrs.Definer])(nil)
	_ views.MethodsView  = (*ChooserListPage[attrs.Definer])(nil)
	_ views.BindableView = (*ChooserListPage[attrs.Definer])(nil)
	_ views.Renderer     = (*BoundChooserListPage[attrs.Definer])(nil)
)

type Renderable interface {
	Render() string
}

type ChooserListPage[T attrs.Definer] struct {
	Template       string
	SearchQueryVar string
	AllowedMethods []string

	// Fields to include for the model in the view
	Fields []string

	// Labels for the fields in the view
	//
	// This is a map of field name to a function that returns the label for the field.
	//
	// Allowing for custom labels for fields in the view.
	Labels map[string]func(ctx context.Context) string

	// PerPage is the number of items to show per page.
	//
	// This is used for pagination in the list view.
	PerPage uint64

	// Columns are used to define the columns in the list view.
	//
	// This allows for custom rendering logic of the columns in the list view.
	Columns map[string]list.ListColumn[T]

	// Format is a map of field name to a function that formats the field value.
	//
	// I.E. map[string]func(v any) any{"Name": func(v any) any { return strings.ToUpper(v.(string)) }}
	// would uppercase the value of the "Name" field in the list view.
	Format map[string]func(v any) any

	// QuerySet is a function that returns a queries.QuerySet to use for the list view.
	QuerySet func(r *http.Request, model T) *queries.QuerySet[T]

	// SearchFields are the fields to search in the list view.
	SearchFields []admin.SearchField

	// NewList returns a new list object to render in the list view.
	NewList func(req *http.Request, results []T, def *ChooserDefinition[T]) any

	// BoundView returns a new bound view for the list page.
	BoundView func(w http.ResponseWriter, req *http.Request, v *ChooserListPage[T], d *ChooserDefinition[T]) (views.View, error)

	_Definition *ChooserDefinition[T]
}

func (v *ChooserListPage[T]) SearchVar() string {
	if v.SearchQueryVar != "" {
		return v.SearchQueryVar
	}
	return "search"
}

func (v *ChooserListPage[T]) ServeXXX(w http.ResponseWriter, req *http.Request) {
	// Placeholder method, will never get called.
}

func (v *ChooserListPage[T]) Methods() []string {
	return v.AllowedMethods
}

func (v *ChooserListPage[T]) Bind(w http.ResponseWriter, req *http.Request) (views.View, error) {
	if v.BoundView != nil {
		return v.BoundView(w, req, v, v._Definition)
	}

	var base = &BoundChooserListPage[T]{
		View:           v,
		ResponseWriter: w,
		Request:        req,
		Model: attrs.NewObject[T](
			reflect.TypeOf(v._Definition.Model),
		),
	}

	return base, nil
}

func (v *ChooserListPage[T]) GetTemplate(r *http.Request) string {
	if v.Template != "" {
		return v.Template
	}

	return "chooser/views/list.tmpl"
}

func (v *ChooserListPage[T]) getQuerySet(req *http.Request) *queries.QuerySet[T] {
	if v.QuerySet != nil {
		return v.QuerySet(req, v._Definition.Model)
	}
	var newObj = attrs.NewObject[T](
		reflect.TypeOf(v._Definition.Model),
	)
	return queries.GetQuerySet(newObj)
}

func (v *ChooserListPage[T]) GetQuerySet(req *http.Request) *queries.QuerySet[T] {
	var qs = v.getQuerySet(req)
	qs = qs.WithContext(req.Context())
	return v.FilterQuerySet(qs, req)
}

func (v *ChooserListPage[T]) FilterQuerySet(qs *queries.QuerySet[T], req *http.Request) *queries.QuerySet[T] {
	if len(v.SearchFields) == 0 {
		return qs
	}

	var searchValue = req.URL.Query().Get(v.SearchVar())
	if searchValue == "" {
		return qs
	}

	var orExprs = make([]expr.Expression, 0, len(v.SearchFields))
	for _, field := range v.SearchFields {

		var expression = field.AsExpression(req, searchValue)
		if expression == nil {
			continue
		}

		orExprs = append(
			orExprs,
			expression,
		)
	}

	if len(orExprs) == 0 {
		return qs
	}

	return qs.Filter(
		expr.Or(orExprs...),
	)
}

func (v *ChooserListPage[T]) ColumnFormat(field string) any {

	if v.Format == nil {
		return field
	}

	var format, ok = v.Format[field]
	if !ok {
		return field
	}

	return func(_ *http.Request, defs attrs.Definitions, _ T) interface{} {
		var value = defs.Get(field)
		return format(value)
	}

}

func (v *ChooserListPage[T]) GetListColumns(req *http.Request) []list.ListColumn[T] {
	var columns = make([]list.ListColumn[T], len(v.Fields))
	for i, field := range v.Fields {
		if v.Columns != nil {
			var col, ok = v.Columns[field]
			if ok {
				columns[i] = col
				continue
			}
		}

		columns[i] = list.Column[T](
			v._Definition.GetLabel(v.Labels, field, field),
			v.ColumnFormat(field),
		)
	}
	return columns
}

func (v *ChooserListPage[T]) GetList(req *http.Request, amount, page int) (any, pagination.PageObject[T], error) {
	var querySet = v.GetQuerySet(req)
	var paginator = &pagination.QueryPaginator[T]{
		URL:     req.URL.Path,
		Context: req.Context(),
		Amount:  int(amount),
		BaseQuerySet: func() *queries.QuerySet[T] {
			return querySet
		},
	}

	var pageObject, err = paginator.Page(page)
	if err != nil && !errors.Is(err, errors.NoRows) {
		return nil, nil, errors.Wrapf(
			err, "failed to get page %d objects with amount %d",
			page, amount,
		)
	}

	//	var listObject = list.NewListWithGroups(req, pageObject.Results(), v.GetListColumns(req), func(r *http.Request, obj T, cols []list.ListColumn[T]) list.ColumnGroup[T] {
	//		return &wrappedColumnGroup[T]{
	//			ListColumnGroup: list.NewColumnGroup(r, obj, cols),
	//			_Definition:     v._Definition,
	//		}
	//	})

	return v.getList(req, pageObject.Results()), pageObject, nil
}

func (v *ChooserListPage[T]) getList(req *http.Request, results []T) any {
	if v.NewList != nil {
		return v.NewList(req, results, v._Definition)
	}

	var listObject = list.NewListWithGroups[T](req, v._Definition.AdminModel.NewInstance().(T), results, v.GetListColumns(req), func(r *http.Request, obj T, cols []list.ListColumn[T]) list.ColumnGroup[T] {
		return &wrappedColumnGroup[T]{
			ListColumnGroup: list.NewColumnGroup(r, obj, cols),
			_Definition:     v._Definition,
		}
	})

	return listObject
}

func (v *ChooserListPage[T]) getPageNumber(req *http.Request) int {
	var pageValue = req.URL.Query().Get("page")
	if pageValue == "" {
		return 1
	}

	var page, err = strconv.Atoi(pageValue)
	if err != nil || page < 1 {
		return 1
	}

	return page
}

func (v *ChooserListPage[T]) GetContext(req *http.Request, bound *BoundChooserListPage[T]) *ModalContext {
	var page = v.getPageNumber(req)
	var listObj, pageObject, err = v.GetList(req, int(v.PerPage), page)
	if err != nil && !errors.Is(err, errors.NoResults) {
		except.Fail(
			http.StatusInternalServerError,
			"Failed to get list for chooser page: %v", err,
		)
		return nil
	}

	if attrSetter, ok := pageObject.(components.AttributeSetter); ok {
		attrSetter.WithAttrs(map[string]any{
			"class": "chooser-link",
		})
	}

	var c = v._Definition.GetContext(req, v, bound)
	c.Set("list", listObj)
	c.Set("page_object", pageObject)
	c.Set("search", map[string]any{
		"var":         v.SearchVar(),
		"allowed":     len(v.SearchFields) > 0,
		"value":       req.URL.Query().Get(v.SearchVar()),
		"placeholder": v._Definition.GetLabel(v.Labels, fmt.Sprintf("%s...", v.SearchVar()), trans.S("Search..."))(req.Context()),
		"text":        v._Definition.GetLabel(v.Labels, v.SearchVar(), trans.S("Search"))(req.Context()),
	})
	return c
}

type BoundChooserListPage[T attrs.Definer] struct {
	View           *ChooserListPage[T]
	ResponseWriter http.ResponseWriter
	Request        *http.Request
	Model          T
}

func (v *BoundChooserListPage[T]) Setup(w http.ResponseWriter, req *http.Request) (http.ResponseWriter, *http.Request) {
	v.ResponseWriter = w
	v.Request = req

	if !permissions.HasObjectPermission(req, v.Model, "admin:view_list") {
		except.Fail(
			http.StatusForbidden,
			"User does not have permission to view this object",
		)
		return nil, nil
	}

	return w, req
}

func (v *BoundChooserListPage[T]) ServeXXX(w http.ResponseWriter, req *http.Request) {
	// Placeholder method, will never get called.
}

func (v *BoundChooserListPage[T]) GetContext(req *http.Request) (ctx.Context, error) {
	var c = v.View.GetContext(req, v)

	return c, nil
}

func (v *BoundChooserListPage[T]) Render(w http.ResponseWriter, req *http.Request, context ctx.Context) error {
	return v.View._Definition.Render(w, req, context, "", v.View.GetTemplate(req))
}
