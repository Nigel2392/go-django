package payments

import (
	"context"
	"iter"
	"strings"

	"github.com/Nigel2392/go-django/contrib/shop/shared/price"
	"github.com/elliotchance/orderedmap/v2"
	"github.com/shopspring/decimal"
	"golang.org/x/text/currency"
)

var registry = &providerRegistry{
	backends: orderedmap.NewOrderedMap[string, ProviderBackend](),
}

// ProviderBackend defines a registered payment service provider (PSP) configuration
// and adapter factory kept strictly in Go code outside the database.
type ProviderBackend interface {
	Ordering() int // lowest first, defaults to InternalName otherwise
	InternalName() string
	Label(ctx context.Context) string
	Currency(ctx context.Context) currency.Unit
	FixedFee(ctx context.Context) decimal.Decimal
	PercentageFee(ctx context.Context) decimal.Decimal
	Provider(ctx context.Context, account *ProviderAccount) (Provider, error)
}

// RegisterProvider registers (or replaces) a PSP backend keyed by its InternalName.
func RegisterProvider(b ProviderBackend) {
	if b == nil || strings.TrimSpace(b.InternalName()) == "" {
		panic("payments: provider backend and non-empty internal name are required")
	}
	if b.InternalName() == freeCheckoutProvider {
		panic("payments: reserved provider key " + b.InternalName())
	}
	registry.register(b)
}

// RetrieveProvider looks up a registered PSP backend by its internal name.
func RetrieveProvider[S ~string](name S) (ProviderBackend, bool) {
	return registry.retrieve(string(name))
}

// ProviderRegistered reports whether a PSP backend with the given internal name is registered.
func ProviderRegistered[S ~string](name S) bool {
	_, ok := registry.retrieve(string(name))
	return ok
}

// ProviderBackendCount returns the number of registered PSP backends.
func ProviderBackendCount() int {
	return registry.count()
}

// IterProviderBackends iterates over registered PSP backends, optionally sorted by Ordering then InternalName.
func IterProviderBackends(useOrdering bool) iter.Seq2[int, ProviderBackend] {
	return registry.iter(useOrdering)
}

// BoundProviderBackends returns all registered PSP backends bound to ctx in display order.
func BoundProviderBackends(ctx context.Context) []BoundProviderBackend {
	list := make([]BoundProviderBackend, 0, ProviderBackendCount())
	for _, backend := range IterProviderBackends(true) {
		list = append(list, BoundProviderBackend{
			Context:         ctx,
			ProviderBackend: backend,
		})
	}
	return list
}

// BoundProviderBackend pairs a ProviderBackend with a context and optional ProviderAccount row.
type BoundProviderBackend struct {
	Context  context.Context
	Instance *ProviderAccount
	ProviderBackend
}

func (b *BoundProviderBackend) Label() string {
	return b.ProviderBackend.Label(b.Context)
}

func (b *BoundProviderBackend) Currency() currency.Unit {
	return b.ProviderBackend.Currency(b.Context)
}

func (b *BoundProviderBackend) FixedFee() decimal.Decimal {
	return b.ProviderBackend.FixedFee(b.Context)
}

func (b *BoundProviderBackend) PercentageFee() decimal.Decimal {
	return b.ProviderBackend.PercentageFee(b.Context)
}

func (b *BoundProviderBackend) Quote(subtotal price.Money) (price.Quote, error) {
	return quoteBackend(b.Context, b.ProviderBackend, subtotal)
}
