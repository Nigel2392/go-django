package components

import "github.com/elliotchance/orderedmap/v2"

type Item interface {
	Name() string
	Order() int
}

type Items[T Item] interface {
	All() []T
	Append(T)
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
		items[idx] = front.Value
		idx++
	}
	return items
}

func (i *ComponentList[T]) Append(item T) {
	i.m.Set(item.Name(), item)
}

func (i *ComponentList[T]) Delete(name string) (ok bool) {
	return i.m.Delete(name)
}
