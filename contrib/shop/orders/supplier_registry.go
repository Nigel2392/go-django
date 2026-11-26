package orders

import (
	"cmp"
	"iter"
	"slices"

	"github.com/elliotchance/orderedmap/v2"
)

type backendRegistry struct {
	backends *orderedmap.OrderedMap[string, SupplierBackend]
}

func (b *backendRegistry) register(backend SupplierBackend) {
	b.backends.Set(backend.InternalName(), backend)
}

func (b *backendRegistry) retrieve(name string) (SupplierBackend, bool) {
	return b.backends.Get(name)
}

func (b *backendRegistry) count() int {
	return b.backends.Len()
}

func (b *backendRegistry) iter(useOrdering bool) iter.Seq2[int, SupplierBackend] {
	if !useOrdering {
		return func(yield func(int, SupplierBackend) bool) {
			var i int
			for head := b.backends.Front(); head != nil; head = head.Next() {
				if !yield(i, head.Value) {
					break
				}
				i++
			}
		}
	}

	var list = make([]*orderedmap.Element[string, SupplierBackend], 0, b.count())
	for head := b.backends.Front(); head != nil; head = head.Next() {
		list = append(list, head)
	}

	slices.SortStableFunc(list, func(a, b *orderedmap.Element[string, SupplierBackend]) int {
		if order := cmp.Compare(a.Value.Ordering(), b.Value.Ordering()); order != 0 {
			return order
		}
		return cmp.Compare(a.Key, b.Key)
	})

	return func(yield func(int, SupplierBackend) bool) {
		for i, elem := range list {
			if !yield(i, elem.Value) {
				break
			}
		}
	}
}
