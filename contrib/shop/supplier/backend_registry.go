package supplier

import (
	"cmp"
	"iter"
	"slices"

	"github.com/elliotchance/orderedmap/v2"
)

type backendRegistry struct {
	backends *orderedmap.OrderedMap[string, Backend]
}

func (b *backendRegistry) register(backend Backend) {
	b.backends.Set(backend.InternalName(), backend)
}

func (b *backendRegistry) retrieve(name string) (Backend, bool) {
	return b.backends.Get(name)
}

func (b *backendRegistry) count() int {
	return b.backends.Len()
}

func (b *backendRegistry) iter(useOrdering bool) iter.Seq2[int, Backend] {
	if !useOrdering {
		return func(yield func(int, Backend) bool) {
			var i int
			for head := b.backends.Front(); head != nil; head = head.Next() {
				if !yield(i, head.Value) {
					break
				}
				i++
			}
		}
	}

	var list = make([]*orderedmap.Element[string, Backend], 0, registry.backends.Len())
	for head := b.backends.Front(); head != nil; head = head.Next() {
		list = append(list, head)
	}

	slices.SortStableFunc(list, func(a, b *orderedmap.Element[string, Backend]) int {
		return slices.Compare(
			[]int{cmp.Compare(a.Value.Ordering(), b.Value.Ordering()), cmp.Compare(a.Key, b.Key)},
			[]int{cmp.Compare(b.Value.Ordering(), a.Value.Ordering()), cmp.Compare(b.Key, a.Key)},
		)
	})

	return func(yield func(int, Backend) bool) {
		for i, elem := range list {
			if !yield(i, elem.Value) {
				break
			}
		}
	}
}
