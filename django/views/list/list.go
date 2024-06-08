package list

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/core/ctx"
	"github.com/Nigel2392/django/views"
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

	if v.TitleFieldColumn != nil {
		cols[0] = v.TitleFieldColumn(cols[0])
	}

	fmt.Println("fields", cols, len(cols))

	list, err := v.GetListFn(uint(amount), uint(page), fields)
	if err != nil {
		return base, err
	}

	listObj := NewList[T](list, cols...)

	base.Set("list", listObj)
	base.Set("amount", amount)
	base.Set("page", page)
	base.Set("max_amount", v.MaxAmount)
	base.Set("amount_param", v.AmountParam)
	base.Set("page_param", v.PageParam)

	return base, nil
}

func (v *View[T]) Columns() []ListColumn[T] {
	if v.ListColumns == nil {
		return []ListColumn[T]{}
	}

	return v.ListColumns
}
