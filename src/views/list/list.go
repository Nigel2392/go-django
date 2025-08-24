package list

import (
	"context"
	"net/http"
	"reflect"
	"strconv"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/core/pagination"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/a-h/templ"
)

type listContextKey struct {
	v string
}

var (
	contextKeyPaginator      = &listContextKey{"paginator"}
	contextKeyPage           = &listContextKey{"page"}
	contextKeyAllowEdit      = &listContextKey{"allow_edit"}
	contextKeyAllowRowSelect = &listContextKey{"allow_row_select"}
)

func PaginatorFromContext[T attrs.Definer](ctx context.Context) pagination.Pagination[T] {
	paginator, _ := ctx.Value(contextKeyPaginator).(pagination.Pagination[T])
	return paginator
}

func PageFromContext[T any](ctx context.Context) pagination.PageObject[T] {
	page, _ := ctx.Value(contextKeyPage).(pagination.PageObject[T])
	return page
}

func AllowListEdit(ctx context.Context) bool {
	allowEdit, _ := ctx.Value(contextKeyAllowEdit).(bool)
	return allowEdit
}

func SetAllowListEdit(ctx context.Context, allow bool) context.Context {
	return context.WithValue(ctx, contextKeyAllowEdit, allow)
}

func AllowListRowSelect(ctx context.Context) bool {
	allowRowSelect, _ := ctx.Value(contextKeyAllowRowSelect).(bool)
	return allowRowSelect
}

func SetAllowListRowSelect(ctx context.Context, allow bool) context.Context {
	return context.WithValue(ctx, contextKeyAllowRowSelect, allow)
}

type ListColumn[T attrs.Definer] interface {
	Header() templ.Component
	Component(r *http.Request, defs attrs.Definitions, row T) templ.Component
}

type ListMediaColumn[T attrs.Definer] interface {
	ListColumn[T]
	media.MediaDefiner
}

type ColumnGroup[T attrs.Definer] interface {
	AddColumn(column ListColumn[T])
	Component(r *http.Request) templ.Component
}

type listView__QuerySetGetter[T attrs.Definer] interface {
	GetQuerySet(r *http.Request) (*queries.QuerySet[T], error)
}

type listView__QuerySetFilterer[T attrs.Definer] interface {
	FilterQuerySet(r *http.Request, qs *queries.QuerySet[T]) (*queries.QuerySet[T], error)
}

type listView__Paginator[T attrs.Definer] interface {
	GetPaginator(r *http.Request, qs *queries.QuerySet[T]) (pagination.Pagination[T], pagination.PageObject[T], error)
}

type listView__ContextGetter[T attrs.Definer] interface {
	GetContext(r *http.Request, qs *queries.QuerySet[T], context ctx.Context) (ctx.Context, error)
}

type StringRenderer interface {
	Render() string
}

type listView__ColumnGetter[T attrs.Definer] interface {
	GetListColumns(r *http.Request) ([]ListColumn[T], error)
}

type listView__ListGetter[T attrs.Definer] interface {
	GetList(r *http.Request, pageObject pagination.PageObject[T], columns []ListColumn[T], context ctx.Context) (StringRenderer, error)
}

var (
	_ listView__ColumnGetter[attrs.Definer]   = (*View[attrs.Definer])(nil)
	_ listView__ListGetter[attrs.Definer]     = (*View[attrs.Definer])(nil)
	_ listView__QuerySetGetter[attrs.Definer] = (*View[attrs.Definer])(nil)
	_ listView__Paginator[attrs.Definer]      = (*View[attrs.Definer])(nil)
	_ views.View                              = (*View[attrs.Definer])(nil)
	_ views.ControlledView                    = (*View[attrs.Definer])(nil)
)

type View[T attrs.Definer] struct {
	AllowedMethods   []string
	BaseTemplateKey  string
	TemplateName     string
	AmountParam      string
	PageParam        string
	MaxAmount        int
	DefaultAmount    int
	ListColumns      []ListColumn[T]
	TitleFieldColumn func(col ListColumn[T]) ListColumn[T]
	GetContextFn     func(r *http.Request, qs *queries.QuerySet[T]) (ctx.Context, error)
	List             func(*http.Request, pagination.PageObject[T], []ListColumn[T], ctx.Context) (StringRenderer, error)
	QuerySet         func(r *http.Request) *queries.QuerySet[T]
	OnError          func(w http.ResponseWriter, r *http.Request, err error)
}

func (v *View[T]) onError(w http.ResponseWriter, r *http.Request, err error) {
	logger.Errorf(
		"Error while serving view: %v", err,
	)
	if v.OnError != nil {
		v.OnError(w, r, err)
	} else {
		except.Fail(
			http.StatusInternalServerError,
			"Error while serving view: %v", err,
		)
	}
}

func (v *View[T]) ServeXXX(w http.ResponseWriter, req *http.Request) {}

func (v *View[T]) Methods() []string {
	return v.AllowedMethods
}

func (v *View[T]) GetTemplate(req *http.Request) string {
	return v.TemplateName
}

func (v *View[T]) Render(w http.ResponseWriter, req *http.Request, templateName string, context ctx.Context) error {
	return tpl.FRender(w, context, v.BaseTemplateKey, templateName)
}

func (v *View[T]) GetListColumns(r *http.Request) ([]ListColumn[T], error) {
	if v.ListColumns == nil {
		return nil, errors.ValueError.Wrapf("no columns defined for list view")
	}

	var cols = v.ListColumns
	if v.TitleFieldColumn != nil && len(cols) > 0 {
		cols[0] = v.TitleFieldColumn(cols[0])
	}

	return cols, nil
}

func (v *View[T]) GetList(r *http.Request, pageObject pagination.PageObject[T], columns []ListColumn[T], context ctx.Context) (list StringRenderer, err error) {
	if v.List != nil {
		list, err = v.List(r, pageObject, columns, context)
	}
	if err != nil {
		return nil, err
	}

	if list == nil {
		var results []T
		if pageObject != nil {
			results = pageObject.Results()
		}
		list = NewList(r, results, columns...)
	}

	return list, nil
}

func (v *View[T]) GetQuerySet(r *http.Request) (*queries.QuerySet[T], error) {
	var qs *queries.QuerySet[T]
	if v.QuerySet == nil {
		var newObj = attrs.NewObject[T](
			reflect.TypeOf(new(T)).Elem(),
		)
		qs = queries.GetQuerySet(newObj)
	} else {
		qs = v.QuerySet(r)
	}
	return qs, nil
}

func (v *View[T]) GetContext(r *http.Request, qs *queries.QuerySet[T]) (ctx.Context, error) {
	if v.GetContextFn != nil {
		return v.GetContextFn(r, qs)
	}
	return ctx.RequestContext(r), nil
}

func (v *View[T]) GetPaginator(req *http.Request, qs *queries.QuerySet[T]) (pagination.Pagination[T], pagination.PageObject[T], error) {
	var (
		amountValue = req.URL.Query().Get(v.AmountParam)
		pageValue   = req.URL.Query().Get(v.PageParam)
		amount      int
		page        int
		err         error
	)
	if amountValue == "" {
		amount = v.DefaultAmount
	} else {
		amount, err = strconv.Atoi(amountValue)
	}
	if err != nil {
		amount = v.DefaultAmount
	}

	if pageValue == "" {
		page = 1
	} else {
		page, err = strconv.Atoi(pageValue)
	}
	if err != nil {
		page = 1
	}

	var paginator = &pagination.QueryPaginator[T]{
		Context: req.Context(),
		Amount:  int(amount),
		BaseQuerySet: func() *queries.QuerySet[T] {
			return qs
		},
	}

	var pageObject pagination.PageObject[T]
	pageObject, err = paginator.Page(int(page))
	return paginator, pageObject, err
}

func (v *View[T]) TakeControl(w http.ResponseWriter, r *http.Request, view views.View) {

	var err error
	var qs *queries.QuerySet[T]
	if getter, ok := view.(listView__QuerySetGetter[T]); ok {
		qs, err = getter.GetQuerySet(r)
	} else {
		qs, err = v.GetQuerySet(r)
	}
	if err != nil {
		v.onError(w, r, err)
		return
	}

	qs = qs.WithContext(r.Context())

	if getter, ok := view.(listView__QuerySetFilterer[T]); ok {
		qs, err = getter.FilterQuerySet(r, qs)
		if err != nil {
			v.onError(w, r, err)
			return
		}
	}

	var (
		paginator  pagination.Pagination[T]
		pageObject pagination.PageObject[T]
	)
	if getter, ok := view.(listView__Paginator[T]); ok {
		paginator, pageObject, err = getter.GetPaginator(r, qs)
	} else {
		paginator, pageObject, err = v.GetPaginator(r, qs)
	}
	if err != nil && !errors.Is(err, errors.NoRows) {
		return
	}

	r = r.WithContext(
		context.WithValue(
			context.WithValue(r.Context(), contextKeyPaginator, paginator),
			contextKeyPage, pageObject,
		),
	)

	viewCtx, err := v.GetContext(r, qs)
	if err != nil {
		v.onError(w, r, err)
		return
	}

	viewCtx.Set("view", view)
	viewCtx.Set("view_paginator", paginator)
	viewCtx.Set("view_paginator_object", pageObject)
	viewCtx.Set("view_max_amount", v.MaxAmount)
	viewCtx.Set("view_amount_param", v.AmountParam)
	viewCtx.Set("view_page_param", v.PageParam)

	if listGetter, ok := view.(listView__ListGetter[T]); ok {
		var columns []ListColumn[T]
		if colGetter, ok := view.(listView__ColumnGetter[T]); ok {
			columns, err = colGetter.GetListColumns(r)
		} else {
			columns, err = v.GetListColumns(r)
		}
		if err != nil {
			v.onError(w, r, err)
			return
		}

		var list StringRenderer
		list, err = listGetter.GetList(r, pageObject, columns, viewCtx)
		if err != nil {
			v.onError(w, r, err)
			return
		}

		viewCtx.Set("view_list", list)
	}

	if getter, ok := view.(listView__ContextGetter[T]); ok {
		viewCtx, err = getter.GetContext(r, qs, viewCtx)
		if err != nil {
			v.onError(w, r, err)
			return
		}
	}

	if err = views.TryServeTemplateView(w, r, view, viewCtx); err != nil {
		v.onError(w, r, err)
	}
}

//func Column[T any](header string, data func(row interface{}) string) ListColumn[T] {
//	return &column[T]{header, data}
//}
