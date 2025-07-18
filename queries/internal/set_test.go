package internal_test

import (
	"testing"

	"github.com/Nigel2392/go-django/queries/internal"
)

var _ internal.Set[any] = (*SliceOrderedSet[any])(nil)

type SliceOrderedSet[E comparable] struct {
	items []E
	index map[E]int
}

func NewSliceOrderedSet[E comparable](elements ...E) *SliceOrderedSet[E] {
	s := &SliceOrderedSet[E]{
		index: make(map[E]int, len(elements)),
	}
	for _, e := range elements {
		s.Add(e)
	}
	return s
}

func (s *SliceOrderedSet[E]) Add(elems ...E) {
	for _, e := range elems {
		if _, exists := s.index[e]; exists {
			continue
		}
		s.items = append(s.items, e)
		s.index[e] = len(s.items) - 1
	}
}

func (s *SliceOrderedSet[E]) Remove(e E) {
	pos, ok := s.index[e]
	if !ok {
		return
	}
	delete(s.index, e)

	// Shift all elements left to maintain order
	copy(s.items[pos:], s.items[pos+1:])
	s.items = s.items[:len(s.items)-1]

	// Rebuild index
	for i := pos; i < len(s.items); i++ {
		s.index[s.items[i]] = i
	}
}

func (s *SliceOrderedSet[E]) Contains(e E) bool {
	_, ok := s.index[e]
	return ok
}

func (s *SliceOrderedSet[E]) Clear() {
	s.items = nil
	s.index = make(map[E]int)
}

func (s *SliceOrderedSet[E]) Len() int {
	return len(s.items)
}

func (s *SliceOrderedSet[E]) Values() []E {
	return append([]E(nil), s.items...) // copy to avoid external mutation
}

const N = 10000

func genInts(n int) []int {
	var out = make([]int, n)
	for i := range out {
		out[i] = i
	}
	return out
}

func BenchmarkLinkedOrderedSet_Add(b *testing.B) {
	data := genInts(N)
	for i := 0; i < b.N; i++ {
		s := internal.NewOrderedSet[int]()
		s.Add(data...)
	}
}

func BenchmarkLinkedOrderedSet_Contains(b *testing.B) {
	data := genInts(N)
	s := internal.NewOrderedSet[int](data...)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.Contains(i % N)
	}
}

func BenchmarkLinkedOrderedSet_Remove(b *testing.B) {
	data := genInts(N)
	for i := 0; i < b.N; i++ {
		s := internal.NewOrderedSet[int](data...)
		for _, x := range data {
			s.Remove(x)
		}
	}
}

func BenchmarkLinkedOrderedSet_Values(b *testing.B) {
	data := genInts(N)
	s := internal.NewOrderedSet[int](data...)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.Values()
	}
}

func BenchmarkSliceOrderedSet_Add(b *testing.B) {
	data := genInts(N)
	for i := 0; i < b.N; i++ {
		s := NewSliceOrderedSet[int]()
		s.Add(data...)
	}
}

func BenchmarkSliceOrderedSet_Contains(b *testing.B) {
	data := genInts(N)
	s := NewSliceOrderedSet[int](data...)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.Contains(i % N)
	}
}

func BenchmarkSliceOrderedSet_Remove(b *testing.B) {
	data := genInts(N)
	for i := 0; i < b.N; i++ {
		s := NewSliceOrderedSet[int](data...)
		for _, x := range data {
			s.Remove(x)
		}
	}
}

func BenchmarkSliceOrderedSet_Values(b *testing.B) {
	data := genInts(N)
	s := NewSliceOrderedSet[int](data...)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.Values()
	}
}
