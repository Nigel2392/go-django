package payments

import (
	"context"
	"fmt"

	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/shopspring/decimal"
	"golang.org/x/text/currency"
)

var _ ProviderBackend = (*PSP)(nil)

// PSP is a declarative ProviderBackend implementation whose configuration and
// factory stay in Go code and are never persisted to the database.
type PSP struct {
	Order             int
	Name              string
	TextLabel         func(ctx context.Context) string
	CurrencyUnit      currency.Unit
	GetCurrency       func(ctx context.Context) currency.Unit
	FixedFeeAmount    decimal.Decimal
	GetFixedFee       func(ctx context.Context) decimal.Decimal
	PercentageFeeRate decimal.Decimal
	GetPercentageFee  func(ctx context.Context) decimal.Decimal
	Factory           ProviderFactory
}

// Ordering returns the display priority (lowest first).
func (p *PSP) Ordering() int {
	return p.Order
}

// InternalName returns the unique registry key persisted on ProviderAccount.
func (p *PSP) InternalName() string {
	return p.Name
}

// Label returns the human-readable label, falling back to a formatted Name.
func (p *PSP) Label(ctx context.Context) string {
	if p.TextLabel == nil {
		return attrs.NiceName(p.Name)
	}
	return p.TextLabel(ctx)
}

// Currency returns the developer-controlled ISO currency unit supported by this PSP.
func (p *PSP) Currency(ctx context.Context) currency.Unit {
	if p.GetCurrency != nil {
		return p.GetCurrency(ctx)
	}
	return p.CurrencyUnit
}

// FixedFee returns the customer-facing fixed fee for this PSP.
func (p *PSP) FixedFee(ctx context.Context) decimal.Decimal {
	if p.GetFixedFee != nil {
		return p.GetFixedFee(ctx)
	}
	return p.FixedFeeAmount
}

// PercentageFee returns the customer-facing percentage fee for this PSP.
func (p *PSP) PercentageFee(ctx context.Context) decimal.Decimal {
	if p.GetPercentageFee != nil {
		return p.GetPercentageFee(ctx)
	}
	return p.PercentageFeeRate
}

// Provider instantiates the runtime Provider adapter for the given account.
func (p *PSP) Provider(ctx context.Context, account *ProviderAccount) (Provider, error) {
	if p.Factory == nil {
		return nil, fmt.Errorf("payment provider %q is unavailable", p.Name)
	}
	return p.Factory(ctx, account)
}
