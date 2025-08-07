package chooser

import (
	"context"
	"net/http"
	"reflect"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/pagination"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/go-django/src/views/list"
)

var _ views.View = (*ChooserListPage[attrs.Definer])(nil)

var (
	_ views.View         = (*ChooserListPage[attrs.Definer])(nil)
	_ views.MethodsView  = (*ChooserListPage[attrs.Definer])(nil)
	_ views.BindableView = (*ChooserListPage[attrs.Definer])(nil)
	_ views.Renderer     = (*BoundChooserListPage[attrs.Definer])(nil)
)

type ChooserListPage[T attrs.Definer] struct {
	Template       string
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

	// BoundView returns a new bound view for the list page.
	BoundView func(w http.ResponseWriter, req *http.Request) (views.View, error)

	_Definition *ChooserDefinition[T]
}

func (v *ChooserListPage[T]) ServeXXX(w http.ResponseWriter, req *http.Request) {
	// Placeholder method, will never get called.
}

func (v *ChooserListPage[T]) Methods() []string {
	return v.AllowedMethods
}

func (v *ChooserListPage[T]) Bind(w http.ResponseWriter, req *http.Request) (views.View, error) {
	if v.BoundView != nil {
		return v.BoundView(w, req)
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

	return "chooser/views/list.html"
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
	return qs.WithContext(req.Context())
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

func (v *ChooserListPage[T]) GetList(req *http.Request, amount, page int) (*list.List[T], error) {
	var querySet = v.GetQuerySet(req)
	var paginator = &pagination.QueryPaginator[T]{
		Context: req.Context(),
		Amount:  int(amount),
		BaseQuerySet: func() *queries.QuerySet[T] {
			return querySet
		},
	}

	var objects, err = paginator.Page(page)
	if err != nil {
		return nil, err
	}

	return list.NewList(req, objects.Results(), v.GetListColumns(req)...), nil
}

func (v *ChooserListPage[T]) GetContext(req *http.Request, bound *BoundChooserListPage[T]) *ModalContext {
	var listObj, err = v.GetList(req, int(v.PerPage), 1)
	if err != nil {
		except.Fail(
			http.StatusInternalServerError,
			"Failed to get list for chooser page: %v", err,
		)
		return nil
	}

	var c = v._Definition.GetContext(req, v, bound)
	c.Set("list", listObj)
	return c
}

type BoundChooserListPage[T attrs.Definer] struct {
	View           *ChooserListPage[T]
	ResponseWriter http.ResponseWriter
	Request        *http.Request
	Model          T
}

func (v *BoundChooserListPage[T]) ServeXXX(w http.ResponseWriter, req *http.Request) {
	// Placeholder method, will never get called.
}

func (v *BoundChooserListPage[T]) GetContext(req *http.Request) (ctx.Context, error) {
	var c = v.View.GetContext(req, v)

	return c, nil
}

func (v *BoundChooserListPage[T]) Render(w http.ResponseWriter, req *http.Request, context ctx.Context) error {
	return nil
}
