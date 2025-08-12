package pagination

import (
	"context"
	"reflect"
	"strconv"
)

type PageObject[T any] interface {
	// The number of objects on this page.
	//
	// This is NOT the total number of objects in the paginator.
	//
	// This might be less than the per page amount
	Count() int

	// A list of results on this page
	// Results() iter.Seq[S, E]
	Results() []T

	// The current page number
	PageNum() int

	// If the page has a next page
	HasNext() bool

	// If the page has a previous page
	HasPrev() bool

	// The next page number
	// This should return -1 if there is no next page
	Next() int

	// The previous page number
	// This should return -1 if there is no previous page
	Prev() int

	// A backreference to the paginator
	Paginator() Pagination[T]
}

type Pagination[T any] interface {
	// The amount of objects total
	// This is used to calculate the number of pages
	Count() (int, error)
	Page(n int) (PageObject[T], error)
	NumPages() (int, error)
	PerPage() int
}

func GetPageNum(n any) int {
	var v = reflect.ValueOf(n)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return int(v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int(v.Uint())
	case reflect.String:
		var i, err = strconv.Atoi(v.String())
		if err != nil {
			return 0
		}
		return i
	}
	return 0
}

func NewPageObject[T any](ctx context.Context, paginator Pagination[T], num int, results []T) PageObject[T] {
	return &pageObject[T]{num: num - 1, results: results, paginator: paginator, context: ctx}
}

func nullPageObject[T any](ctx context.Context, p Pagination[T]) PageObject[T] {
	return &pageObject[T]{
		paginator: p,
		num:       0,
		results:   make([]T, 0),
		context:   ctx,
	}
}
