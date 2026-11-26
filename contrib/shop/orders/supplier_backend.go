package orders

import (
	"context"
	"html/template"
	"iter"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/blocks"
	"github.com/Nigel2392/go-django/contrib/shop/products"
	"github.com/elliotchance/orderedmap/v2"
)

var registry = &backendRegistry{
	backends: orderedmap.NewOrderedMap[string, SupplierBackend](),
}

type SupplierBackend interface {
	Ordering() int // lowest first, defaults to InternalName otherwise
	InternalName() string
	Icon(ctx context.Context) template.HTML
	Label(ctx context.Context) string
	Description(ctx context.Context) string

	ProductType(ctx context.Context) products.ProductModel
	ProductAddPanel(ctx context.Context) []admin.Panel
	ProductEditPanel(ctx context.Context) []admin.Panel

	OrderBlock(ctx context.Context, order *Order, orderItem *OrderItem) (blocks.Block, error)
	OrderAccepted(ctx context.Context, order *Order, orderItem *OrderItem) (*blocks.StructBlockValue, error)
}

type CatalogableSupplierBackend interface {
	Catalog(ctx context.Context) ([]products.ProductModel, error)
}

func RegisterSupplier(b SupplierBackend) {
	registry.register(b)
}

func RetrieveSupplier[S ~string](name S) (SupplierBackend, bool) {
	return registry.retrieve(string(name))
}

func SupplierBackendCount() int {
	return registry.count()
}

func IterSupplierBackends(useOrdering bool) iter.Seq2[int, SupplierBackend] {
	return registry.iter(useOrdering)
}

func BoundSupplierBackends(ctx context.Context) []BoundSupplierBackend {
	var l = make([]BoundSupplierBackend, 0, SupplierBackendCount())
	for _, backend := range IterSupplierBackends(true) {
		l = append(l, BoundSupplierBackend{
			Context:         ctx,
			SupplierBackend: backend,
		})
	}
	return l
}

type BoundSupplierBackend struct {
	Context  context.Context
	Instance *SupplierModel
	SupplierBackend
}

func (b *BoundSupplierBackend) Icon() template.HTML {
	return b.SupplierBackend.Icon(b.Context)
}

func (b *BoundSupplierBackend) Label() string {
	return b.SupplierBackend.Label(b.Context)
}

func (b *BoundSupplierBackend) Description() string {
	return b.SupplierBackend.Description(b.Context)
}
