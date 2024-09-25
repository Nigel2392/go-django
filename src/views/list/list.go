package list

import (
	"net/http"
	"strconv"

	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/a-h/templ"
)

type ListColumn[T attrs.Definer] interface {
	Header() templ.Component
	Component(defs attrs.Definitions, row T) templ.Component
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
	GetListFn        func(amount, offset uint, include []string) ([]T, error)
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
		page = 0
	} else {
		page, err = strconv.ParseUint(pageValue, 10, 64)
	}
	if err != nil {
		return base, err
	}

	var cols = v.Columns()
	var fields = make([]string, 0, len(cols))
	for _, col := range cols {
		if namer, ok := col.(attrs.Namer); ok {
			fields = append(fields, namer.Name())
		}
	}

	if v.TitleFieldColumn != nil && len(cols) > 0 {
		cols[0] = v.TitleFieldColumn(cols[0])
	}

	list, err := v.GetListFn(uint(amount), uint(page), fields)
	if err != nil {
		return base, err
	}

	listObj := NewList(list, cols...)

	base.Set("view_list", listObj)
	base.Set("view_amount", amount)
	base.Set("view_page", page)
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
