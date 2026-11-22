package payments

import (
	"context"
	"fmt"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
	"github.com/shopspring/decimal"
	"golang.org/x/text/currency"
)

// ProviderAccount configures one merchant account. CredentialRef identifies
// a secret resolved by the provider factory.
type ProviderAccount struct {
	ID            drivers.ULID
	Driver        string
	Label         string
	Enabled       bool
	CredentialRef string `json:"-"`
	Currency      string
	FixedFee      decimal.Decimal
	PercentageFee decimal.Decimal
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
		paymentField(a, "Driver", &a.Driver, attrs.Config{MaxLength: 75}),
		paymentField(a, "Label", &a.Label, attrs.Config{MaxLength: 100}),
		paymentField(a, "Enabled", &a.Enabled, attrs.Config{Blank: true}),
		paymentField(a, "CredentialRef", &a.CredentialRef, attrs.Config{MaxLength: 255, Blank: true}),
		paymentField(a, "Currency", &a.Currency, attrs.Config{MaxLength: 3}),
		paymentField(a, "FixedFee", &a.FixedFee, attrs.Config{Precision: 13, Scale: 4, Blank: true}),
		paymentField(a, "PercentageFee", &a.PercentageFee, attrs.Config{Precision: 9, Scale: 4, Blank: true}),
	)
}

func paymentField[M attrs.Definer, V any](model M, name string, value *V, config attrs.Config) attrs.Field {
	return fattrs.Field(model, name, value, func() fattrs.PtrFieldConfig[M, V] {
		return fattrs.PtrFieldConfig[M, V]{Config: config}
	})
}

type Quote struct {
	Subtotal decimal.Decimal `json:"subtotal"`
	Fee      decimal.Decimal `json:"fee"`
	Total    decimal.Decimal `json:"total"`
	Currency string          `json:"currency"`
}

// Quote adds the account's customer-facing fee to the discounted, tax-inclusive cart total.
func (a *ProviderAccount) Quote(subtotal Money) (Quote, error) {
	unit, err := currency.ParseISO(subtotal.Currency)
	if err != nil || unit.String() != subtotal.Currency || a.Currency != subtotal.Currency {
		return Quote{}, fmt.Errorf("payment account does not support currency %q", subtotal.Currency)
	}
	if subtotal.Amount.IsNegative() || a.FixedFee.IsNegative() || a.PercentageFee.IsNegative() {
		return Quote{}, fmt.Errorf("payment amount and fees must be nonnegative")
	}
	scale, _ := currency.Standard.Rounding(unit)
	base := subtotal.Amount.Round(int32(scale))
	fee := decimal.Zero
	if base.IsPositive() {
		fee = a.FixedFee.Add(base.Mul(a.PercentageFee).Shift(-2)).Round(int32(scale))
	}
	return Quote{Subtotal: base, Fee: fee, Total: base.Add(fee), Currency: subtotal.Currency}, nil
}
