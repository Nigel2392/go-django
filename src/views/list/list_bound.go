package list

import (
	"net/http"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/pagination"
	"github.com/Nigel2392/go-django/src/views"
)

var (
	_ listView__QuerySetGetter[attrs.Definer] = (*BoundView[attrs.Definer])(nil)
	_ listView__Paginator[attrs.Definer]      = (*BoundView[attrs.Definer])(nil)
	_ listView__ContextGetter[attrs.Definer]  = (*BoundView[attrs.Definer])(nil)
	_ listView__ContextGetter[attrs.Definer]  = (*BoundView[attrs.Definer])(nil)
	_ listView__ListGetter[attrs.Definer]     = (*BoundView[attrs.Definer])(nil)
)

type BoundView[T attrs.Definer] struct {
	View       *View[T]
	Bases      []views.View
	W          http.ResponseWriter
	R          *http.Request
	Context    ctx.Context
	QuerySet   *queries.QuerySet[T]
	Pagination pagination.Pagination[T]
	PageObject pagination.PageObject[T]
	Page       int
	Amount     int
}

func NewBoundView[T attrs.Definer](w http.ResponseWriter, r *http.Request, view *View[T], bases []views.View) *BoundView[T] {
	return &BoundView[T]{
		View:  view,
		Bases: bases,
		W:     w,
		R:     r,
	}
}

func (v *BoundView[T]) ServeXXX(w http.ResponseWriter, req *http.Request) {}

func (v *BoundView[T]) Mixins() []views.View {
	return v.Bases
}

func (v *BoundView[T]) GetQuerySet(r *http.Request) (*queries.QuerySet[T], error) {
	var qs, err = v.View.GetQuerySet(r)
	if err != nil {
		return nil, err
	}
	v.QuerySet = qs
	return qs, nil
}

func (v *BoundView[T]) GetPaginator(req *http.Request, qs *queries.QuerySet[T]) (pagination.Pagination[T], pagination.PageObject[T], error) {
	var p, obj, amount, page, err = v.View.getPaginator(req, qs)
	if err != nil && !errors.Is(err, errors.NoRows) {
		return p, obj, err
	}
	v.Amount = amount
	v.Page = page
	v.Pagination = p
	v.PageObject = obj
	return p, obj, nil
}

func (v *BoundView[T]) GetContext(r *http.Request, qs *queries.QuerySet[T], context ctx.Context) (ctx.Context, error) {
	if v.View.ChangeContextFn != nil {
		var err error
		context, err = v.View.ChangeContextFn(r, qs, context)
		if err != nil {
			return context, err
		}
	}
	v.Context = context
	return context, nil
}

func (v *BoundView[T]) GetList(r *http.Request, pageObject pagination.PageObject[T], columns []ListColumn[T], context ctx.Context) (list StringRenderer, err error) {
	if v.View.List != nil {
		list, err = v.View.List(r, pageObject, columns, context)
	}
	if err != nil {
		return nil, err
	}

	if list == nil {
		var results []T
		if pageObject != nil {
			results = pageObject.Results()
		}
		list = NewList(r, v.View.Model, results, columns...)
	}

	return list, nil
}
