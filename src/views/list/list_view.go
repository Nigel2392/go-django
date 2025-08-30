package list

import (
	"context"
	"net/http"
	"reflect"
	"slices"
	"strconv"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/core/pagination"
	"github.com/Nigel2392/go-django/src/utils/mixins"
	"github.com/Nigel2392/go-django/src/views"
)

var (
	_ listView__ColumnGetter[attrs.Definer]   = (*View[attrs.Definer])(nil)
	_ listView__QuerySetGetter[attrs.Definer] = (*View[attrs.Definer])(nil)
	_ listView__Paginator[attrs.Definer]      = (*View[attrs.Definer])(nil)
	_ views.View                              = (*View[attrs.Definer])(nil)
	_ views.ControlledView                    = (*View[attrs.Definer])(nil)
)

type View[T attrs.Definer] struct {
	Model            T
	AllowedMethods   []string
	BaseTemplateKey  string
	TemplateName     string
	AmountParam      string
	PageParam        string
	MaxAmount        int
	DefaultAmount    int
	ListColumns      []ListColumn[T]
	Mixins           func(r *http.Request, v *View[T]) []views.View
	TitleFieldColumn func(col ListColumn[T]) ListColumn[T]
	GetContextFn     func(r *http.Request, qs *queries.QuerySet[T]) (ctx.Context, error)
	ChangeContextFn  func(r *http.Request, qs *queries.QuerySet[T], ctx ctx.Context) (ctx.Context, error)
	List             func(*http.Request, pagination.PageObject[T], []ListColumn[T], ctx.Context) (StringRenderer, error)
	QuerySet         func(r *http.Request) *queries.QuerySet[T]
	OnError          func(w http.ResponseWriter, r *http.Request, err error)
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

	for mixin, depth := range mixins.Mixins(view, false) {
		if depth == 0 {
			continue
		}

		if filterer, ok := mixin.(listView__QuerySetFilterer[T]); ok {
			qs, err = filterer.FilterQuerySet(r, qs)
			if err != nil {
				v.onError(w, r, err)
				return
			}
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
		v.onError(w, r, err)
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

	for mixin, depth := range mixins.Mixins(view, false) {
		// only call the GetContext method for mixins.
		// do not include the view itself, the GetContext method
		// of the view will be called later.
		if depth == 0 {
			continue
		}

		if getter, ok := mixin.(listView__ContextGetter[T]); ok {
			viewCtx, err = getter.GetContext(r, qs, viewCtx)
			if err != nil {
				v.onError(w, r, err)
				return
			}
		}

		if getter, ok := mixin.(listView__MixinContextGetter[T]); ok {
			viewCtx, err = getter.GetContext(r, view, qs, viewCtx)
			if err != nil {
				v.onError(w, r, err)
				return
			}
		}
	}

	if getter, ok := view.(listView__ContextGetter[T]); ok {
		viewCtx, err = getter.GetContext(r, qs, viewCtx)
		if err != nil {
			v.onError(w, r, err)
			return
		}
	}

	for mixin, depth := range mixins.Mixins(view, false) {
		// The view itself cannot hijack like this.
		// The view has to override the Render method.
		if depth == 0 {
			continue
		}

		if getter, ok := mixin.(listView__MixinHijacker[T]); ok {
			wr, req, err := getter.Hijack(w, r, view, qs, viewCtx)
			if err != nil {
				v.onError(w, r, err)
				return
			}

			// The mixin has hijacked and written to the response.
			// No need to keep processing further.
			if wr == nil || req == nil {
				return
			}
			w = wr
			r = req
		}
	}

	if err = views.TryServeTemplateView(w, r, []views.View{view, v}, viewCtx); err != nil {
		v.onError(w, r, err)
	}
}

func (v *View[T]) Clone() *View[T] {
	return &View[T]{
		Model:            attrs.NewObject[T](reflect.TypeOf(v.Model)),
		AllowedMethods:   slices.Clone(v.AllowedMethods),
		BaseTemplateKey:  v.BaseTemplateKey,
		TemplateName:     v.TemplateName,
		AmountParam:      v.AmountParam,
		PageParam:        v.PageParam,
		MaxAmount:        v.MaxAmount,
		DefaultAmount:    v.DefaultAmount,
		ListColumns:      slices.Clone(v.ListColumns),
		TitleFieldColumn: v.TitleFieldColumn,
		GetContextFn:     v.GetContextFn,
		List:             v.List,
		QuerySet:         v.QuerySet,
		OnError:          v.OnError,
	}
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

func (v *View[T]) Bind(w http.ResponseWriter, req *http.Request) (views.View, error) {
	var mixins []views.View = []views.View{
		&ListObjectMixin[T]{ListView: v},
	}
	if v.Mixins != nil {
		mixins = append(mixins, v.Mixins(req, v)...)
	}
	return NewBoundView(w, req, v, mixins), nil
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

func (v *View[T]) getPaginator(req *http.Request, qs *queries.QuerySet[T]) (pagination.Pagination[T], pagination.PageObject[T], int, int, error) {
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
	return paginator, pageObject, amount, page, err
}

func (v *View[T]) GetPaginator(req *http.Request, qs *queries.QuerySet[T]) (pagination.Pagination[T], pagination.PageObject[T], error) {
	pagination, pageObject, _, _, err := v.getPaginator(req, qs)
	return pagination, pageObject, err
}
