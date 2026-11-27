package payments

import (
	"cmp"
	"iter"
	"slices"
	"sync"

	"github.com/elliotchance/orderedmap/v2"
)

type providerRegistry struct {
	mu       sync.RWMutex
	backends *orderedmap.OrderedMap[string, ProviderBackend]
}

func (r *providerRegistry) register(backend ProviderBackend) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.backends.Set(backend.InternalName(), backend)
}

func (r *providerRegistry) retrieve(name string) (ProviderBackend, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.backends.Get(name)
}

func (r *providerRegistry) count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.backends.Len()
}

func (r *providerRegistry) iter(useOrdering bool) iter.Seq2[int, ProviderBackend] {
	r.mu.RLock()
	list := make([]*orderedmap.Element[string, ProviderBackend], 0, r.backends.Len())
	for head := r.backends.Front(); head != nil; head = head.Next() {
		list = append(list, head)
	}
	r.mu.RUnlock()

	if useOrdering {
		slices.SortStableFunc(list, func(a, b *orderedmap.Element[string, ProviderBackend]) int {
			if order := cmp.Compare(a.Value.Ordering(), b.Value.Ordering()); order != 0 {
				return order
			}
			return cmp.Compare(a.Key, b.Key)
		})
	}

	return func(yield func(int, ProviderBackend) bool) {
		for i, elem := range list {
			if !yield(i, elem.Value) {
				break
			}
		}
	}
}
