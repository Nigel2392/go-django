package list

import (
	"errors"
	"net/http"
	"reflect"
	"strconv"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/pagination"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/a-h/templ"
)

type ListColumn[T attrs.Definer] interface {
	Header() templ.Component
	Component(r *http.Request, defs attrs.Definitions, row T) templ.Component
}

type ColumnGroup[T attrs.Definer] interface {
	AddColumn(column ListColumn[T])
	Component(r *http.Request) templ.Component
}

//func Column[T any](header string, data func(row interface{}) string) ListColumn[T] {
//	return &column[T]{header, data}
//}

type View[T attrs.Definer] struct {
	views.BaseView
	AmountParam      string
	PageParam        string
	MaxAmount        uint64
	DefaultAmount    uint64
	ListColumns      []ListColumn[T]
	TitleFieldColumn func(ListColumn[T]) ListColumn[T]
	QuerySet         func(r *http.Request) *queries.QuerySet[T]
}

func (v *View[T]) GetQuerySet(r *http.Request) *queries.QuerySet[T] {
	var qs *queries.QuerySet[T]
	if v.QuerySet == nil {
		var newObj = attrs.NewObject[T](
			reflect.TypeOf(new(T)).Elem(),
		)
		qs = queries.GetQuerySet(newObj)
	} else {
		qs = v.QuerySet(r)
	}
	return qs.WithContext(r.Context())
}

func (v *View[T]) GetContext(req *http.Request) (ctx.Context, error) {
	var base, err = v.BaseView.GetContext(req)
	if err != nil {
		return base, err
	}

	var (
		amountValue = req.URL.Query().Get(v.AmountParam)
		pageValue   = req.URL.Query().Get(v.PageParam)
		amount      uint64
		page        uint64
	)
	if amountValue == "" {
		amount = v.DefaultAmount
	} else {
		amount, err = strconv.ParseUint(amountValue, 10, 64)
	}
	if err != nil {
		return base, err
	}

	if pageValue == "" {
		page = 1
	} else {
		page, err = strconv.ParseUint(pageValue, 10, 64)
	}
	if err != nil {
		return base, err
	}

	var paginator = &pagination.QueryPaginator[T]{
		Context: req.Context(),
		Amount:  int(amount),
		BaseQuerySet: func() *queries.QuerySet[T] {
			return v.GetQuerySet(req)
		},
	}

	var cols = v.Columns()
	if v.TitleFieldColumn != nil && len(cols) > 0 {
		cols[0] = v.TitleFieldColumn(cols[0])
	}

	pageObject, err := paginator.Page(int(page))
	if err != nil && !errors.Is(err, pagination.ErrNoResults) {
		return base, err
	}

	var results []T
	if pageObject != nil {
		results = pageObject.Results()
	}

	listObj := NewList(req, results, cols...)

	base.Set("view_list", listObj)
	base.Set("view_amount", amount)
	base.Set("view_page", page)
	base.Set("view_paginator", paginator)
	base.Set("view_paginator_object", pageObject)
	base.Set("view_max_amount", v.MaxAmount)
	base.Set("view_amount_param", v.AmountParam)
	base.Set("view_page_param", v.PageParam)

	return base, nil
}

func (v *View[T]) Columns() []ListColumn[T] {
	if v.ListColumns == nil {
		return []ListColumn[T]{}
	}

	return v.ListColumns
}
