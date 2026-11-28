package payments

import (
	"context"
	"fmt"

	"github.com/Nigel2392/go-django/contrib/shop/core/shared/price"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
	"golang.org/x/text/currency"
)

var (
	_ migrator.IndexDefiner = (*ProviderAccount)(nil)
)

// ProviderAccount tracks which registered PSP backends are enabled in the database.
// Configuration, fees, and credentials stay in Go code via the ProviderBackend registry.
type ProviderAccount struct {
	ID           drivers.ULID
	InternalName string
	Enabled      bool
}

func (a *ProviderAccount) UniqueTogether() [][]string {
	return [][]string{
		{"InternalName"},
	}
}

func (a *ProviderAccount) DatabaseIndexes(obj attrs.Definer) []migrator.Index {
	return []migrator.Index{
		{Identifier: "provider_account_internal_name_idx", Fields: []string{"InternalName"}, Unique: true},
	}
}

func (a *ProviderAccount) BeforeCreate(context.Context) error {
	if a.ID.IsZero() {
		a.ID = drivers.NewULID()
	}
	return nil
}

func (a *ProviderAccount) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, a,
		paymentField(a, "ID", &a.ID, attrs.Config{Primary: true, ReadOnly: true}),
		fattrs.Field(a, "InternalName", &a.InternalName, func() fattrs.PtrFieldConfig[*ProviderAccount, string] {
			return fattrs.PtrFieldConfig[*ProviderAccount, string]{
				Config: attrs.Config{
					MinLength: 2,
					MaxLength: 75,
					Attributes: map[string]interface{}{
						attrs.AttrUniqueKey: true,
					},
				},
				IsZero: func(s string) bool {
					return s == ""
				},
			}
		}),
		paymentField(a, "Enabled", &a.Enabled, attrs.Config{Blank: true, Default: true}),
	)
}

func paymentField[M attrs.Definer, V any](model M, name string, value *V, config attrs.Config) attrs.Field {
	return fattrs.Field(model, name, value, func() fattrs.PtrFieldConfig[M, V] {
		return fattrs.PtrFieldConfig[M, V]{Config: config}
	})
}

// Provider resolves this account's adapter, including optional capabilities.
func (a *ProviderAccount) Provider(ctx context.Context) (Provider, error) {
	return a.provider(ctx)
}

func (a *ProviderAccount) provider(ctx context.Context) (Provider, error) {
	if a == nil {
		return nil, fmt.Errorf("payment provider \"\" is unavailable")
	}
	backend, ok := RetrieveProvider(a.InternalName)
	if !ok {
		return nil, fmt.Errorf("payment provider %q is unavailable", a.InternalName)
	}
	return backend.Provider(ctx, a)
}

// Quote adds the registered backend's customer-facing fee to the discounted, tax-inclusive cart total.
func (a *ProviderAccount) Quote(ctx context.Context, subtotal price.Money) (price.Quote, error) {
	if a == nil {
		return price.Quote{}, fmt.Errorf("payment provider \"\" is unavailable")
	}
	backend, ok := RetrieveProvider(a.InternalName)
	if !ok {
		return price.Quote{}, fmt.Errorf("payment provider %q is unavailable", a.InternalName)
	}
	return quoteBackend(ctx, backend, subtotal)
}

func quoteBackend(ctx context.Context, backend ProviderBackend, subtotal price.Money) (price.Quote, error) {
	if backend == nil {
		return price.Quote{}, fmt.Errorf("payment provider is unavailable")
	}
	unit := backend.Currency(ctx)
	if unit == (currency.Unit{}) || subtotal.Currency == (currency.Unit{}) || unit != subtotal.Currency {
		return price.Quote{}, fmt.Errorf("payment account does not support currency %q", subtotal.Currency)
	}
	return price.QuoteFee(subtotal, backend.FixedFee(ctx), backend.PercentageFee(ctx))
}
