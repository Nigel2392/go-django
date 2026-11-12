package supplier

import (
	"context"
	"html/template"
	"iter"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/elliotchance/orderedmap/v2"
)

var registry = &backendRegistry{
	backends: orderedmap.NewOrderedMap[string, Backend](),
}

type Backend interface {
	Ordering() int // lowest first, defaults to InternalName otherwise
	InternalName() string
	Icon(ctx context.Context) template.HTML
	Label(ctx context.Context) string
	Description(ctx context.Context) string
	ProductType(ctx context.Context) ProductModel
	ProductAddPanel(ctx context.Context) []admin.Panel
	ProductEditPanel(ctx context.Context) []admin.Panel
}

func Register(b Backend) {
	registry.register(b)
}

func Retrieve(name string) (Backend, bool) {
	return registry.retrieve(name)
}

func BackendCount() int {
	return registry.count()
}

func IterBackends(useOrdering bool) iter.Seq2[int, Backend] {
	return registry.iter(useOrdering)
}

func BoundBackends(ctx context.Context) []BoundBackend {
	var l = make([]BoundBackend, 0, BackendCount())
	for _, backend := range IterBackends(true) {
		l = append(l, BoundBackend{
			Context: ctx,
			Backend: backend,
		})
	}
	return l
}

type BoundBackend struct {
	Context  context.Context
	Instance *SupplierModel
	Backend
}

func (b *BoundBackend) Icon() template.HTML {
	return b.Backend.Icon(b.Context)
}

func (b *BoundBackend) Label() string {
	return b.Backend.Label(b.Context)
}

func (b *BoundBackend) Description() string {
	return b.Backend.Description(b.Context)
}
