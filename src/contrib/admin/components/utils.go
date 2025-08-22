package components

import (
	"net/http"
	"reflect"
	"slices"

	"github.com/a-h/templ"
	"github.com/elliotchance/orderedmap/v2"
)

type Item interface {
	Name() string
	Order() int
}

type Items[T Item] interface {
	All() []T
	Append(T)
	Get(name string) (T, bool)
	Delete(name string) (ok bool)
}

type ComponentList[T Item] struct {
	m *orderedmap.OrderedMap[string, T]
}

func NewItems[T Item]() Items[T] {
	return &ComponentList[T]{
		m: orderedmap.NewOrderedMap[string, T](),
	}
}

func (i *ComponentList[T]) All() []T {
	var (
		items = make([]T, i.m.Len())
		idx   = 0
	)

	for front := i.m.Front(); front != nil; front = front.Next() {
		v := front.Value
		items[idx] = v
		idx++
	}
	slices.SortStableFunc(items, func(i, j T) int {
		var a, b = i.Order(), j.Order()
		if a < b {
			return -1
		}
		if a > b {
			return 1
		}
		return 0
	})
	return items
}

func (i *ComponentList[T]) Append(item T) {
	var n = item.Name()
	if n == "" {
		var t = reflect.TypeOf(item)
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		n = t.Name()
	}

	i.m.Set(n, item)
}

func (i *ComponentList[T]) Get(name string) (T, bool) {
	return i.m.Get(name)
}

func (i *ComponentList[T]) Delete(name string) (ok bool) {
	return i.m.Delete(name)
}

type ShowableComponent interface {
	IsShown() bool
	templ.Component
}

type showableComponent struct {
	req     *http.Request
	isShown func(r *http.Request) bool
	templ.Component
}

func NewShowableComponent(req *http.Request, isShown func(r *http.Request) bool, component templ.Component) ShowableComponent {
	return &showableComponent{
		req:       req,
		isShown:   isShown,
		Component: component,
	}
}

func (p *showableComponent) IsShown() bool {
	if p.isShown == nil {
		return true
	}
	return p.isShown(p.req)
}
