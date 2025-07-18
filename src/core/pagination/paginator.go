package pagination

import (
	"errors"
	"reflect"
	"strconv"
)

var (
	ErrNoResults = errors.New("no results found")
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

// Paginator holds pagination logic
type Paginator[S ~[]E, E any] struct {
	GetObject  func(E) E
	GetObjects func(amount int, offset int) (S, error)
	GetCount   func() (int, error)
	Amount     int
	cnt        int
}

func (p *Paginator[S, E]) NumPages() (int, error) {
	count, err := p.GetCount()
	if err != nil {
		return 0, err
	}
	return (count + p.PerPage() - 1) / p.PerPage(), nil
}

func (p *Paginator[S, E]) Count() (int, error) {
	if p.cnt == 0 {
		count, err := p.GetCount()
		if err != nil {
			return 0, err
		}
		p.cnt = count
	}
	return p.cnt, nil
}

func (p *Paginator[S, E]) PerPage() int {
	return p.Amount
}

func (p *Paginator[S, E]) Page(n int) (PageObject[E], error) {
	var amount = p.PerPage()

	if amount == 0 {
		return nil, errors.New("amount of objects per page cannot be 0")
	}

	var offset = (n - 1) * amount
	results, err := p.GetObjects(amount, offset)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, ErrNoResults
	}

	if p.GetObject != nil {
		for i, r := range results {
			results[i] = p.GetObject(r)
		}
	}

	return &pageObject[E]{num: n - 1, results: results, paginator: p}, nil
}
