package internal

import (
	"maps"
	"slices"
)

type Set[E comparable] interface {
	Contains(e E) bool
	Add(e ...E)
	Remove(e E)
	Clear()
	Values() []E
	Len() int
}

var _ Set[any] = (BaseSet[any])(nil)

type BaseSet[E comparable] map[E]struct{}

func NewSet[E comparable](s ...E) BaseSet[E] {
	if len(s) == 0 {
		return make(BaseSet[E])
	}

	var m = make(BaseSet[E], len(s))
	for _, e := range s {
		m[e] = struct{}{}
	}
	return m
}

func (s BaseSet[E]) Contains(e E) bool {
	_, ok := s[e]
	return ok
}

func (s BaseSet[E]) Add(e ...E) {
	for _, elem := range e {
		s[elem] = struct{}{}
	}
}

func (s BaseSet[E]) Remove(e E) {
	delete(s, e)
}

func (s BaseSet[E]) Clear() {
	for e := range s {
		delete(s, e)
	}
}

func (s BaseSet[E]) Len() int {
	return len(s)
}

func (s BaseSet[E]) Values() []E {
	return slices.Collect(maps.Keys(s))
}
