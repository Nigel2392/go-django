//go:build test

package payments

import (
	"context"
	"iter"
	"testing"

	"github.com/Nigel2392/go-django/contrib/shop/shared/price"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/elliotchance/orderedmap/v2"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/currency"
)

func TestProviderRegistry(t *testing.T) {
	previous := registry
	registry = &providerRegistry{backends: orderedmap.NewOrderedMap[string, ProviderBackend]()}
	t.Cleanup(func() { registry = previous })

	require.Panics(t, func() { RegisterProvider(nil) })
	require.Panics(t, func() { RegisterProvider(&PSP{Name: "   "}) })
	require.Panics(t, func() { RegisterProvider(&PSP{Name: freeCheckoutProvider}) })

	RegisterProvider(&PSP{Name: "zulu", Order: 5, CurrencyUnit: currency.EUR})
	RegisterProvider(&PSP{Name: "alpha", Order: 5, CurrencyUnit: currency.EUR})
	RegisterProvider(&PSP{
		Name:              "first",
		Order:             -1,
		CurrencyUnit:      currency.EUR,
		FixedFeeAmount:    decimal.RequireFromString("0.50"),
		PercentageFeeRate: decimal.RequireFromString("2.5"),
	})
	replacement := &PSP{
		Name:         "zulu",
		Order:        5,
		CurrencyUnit: currency.USD,
		TextLabel:    func(context.Context) string { return "Zulu Pay" },
		GetCurrency:  func(context.Context) currency.Unit { return currency.EUR },
		GetFixedFee: func(context.Context) decimal.Decimal {
			return decimal.RequireFromString("0.25")
		},
		GetPercentageFee: func(context.Context) decimal.Decimal {
			return decimal.RequireFromString("1")
		},
	}
	RegisterProvider(replacement)

	require.Equal(t, 3, ProviderBackendCount())
	backend, ok := RetrieveProvider("zulu")
	require.True(t, ok)
	require.Same(t, replacement, backend)
	require.True(t, ProviderRegistered("zulu"))
	require.False(t, ProviderRegistered("unknown"))
	_, ok = RetrieveProvider("unknown")
	require.False(t, ok)

	names := func(backends iter.Seq2[int, ProviderBackend]) []string {
		var out []string
		for index, b := range backends {
			require.Equal(t, len(out), index)
			out = append(out, b.InternalName())
		}
		return out
	}
	require.Equal(t, []string{"zulu", "alpha", "first"}, names(IterProviderBackends(false)))
	require.Equal(t, []string{"first", "alpha", "zulu"}, names(IterProviderBackends(true)))

	ctx := t.Context()
	bound := BoundProviderBackends(ctx)
	require.Len(t, bound, 3)
	require.Equal(t, "First", bound[0].Label())
	require.Equal(t, currency.EUR, bound[0].Currency())
	require.True(t, bound[0].FixedFee().Equal(decimal.RequireFromString("0.50")))
	require.True(t, bound[0].PercentageFee().Equal(decimal.RequireFromString("2.5")))

	require.Equal(t, "Zulu Pay", bound[2].Label())
	require.Equal(t, currency.EUR, bound[2].Currency())
	require.True(t, bound[2].FixedFee().Equal(decimal.RequireFromString("0.25")))
	require.True(t, bound[2].PercentageFee().Equal(decimal.RequireFromString("1")))

	quote, err := bound[0].Quote(price.Money{Amount: decimal.NewFromInt(10), Currency: currency.EUR})
	require.NoError(t, err)
	require.True(t, quote.Subtotal.Equal(decimal.NewFromInt(10)))
	require.True(t, quote.Fee.Equal(decimal.RequireFromString("0.75")))
	require.True(t, quote.Total.Equal(decimal.RequireFromString("10.75")))
	require.Equal(t, currency.EUR, quote.Currency)

	acct := &ProviderAccount{InternalName: "first", Enabled: true}
	acctQuote, err := acct.Quote(ctx, price.Money{Amount: decimal.NewFromInt(10), Currency: currency.EUR})
	require.NoError(t, err)
	require.Equal(t, quote, acctQuote)

	_, err = acct.Quote(ctx, price.Money{Amount: decimal.NewFromInt(10), Currency: currency.USD})
	require.Error(t, err)

	_, err = acct.Provider(ctx)
	require.Error(t, err)

	missingAcct := &ProviderAccount{InternalName: "unknown"}
	_, err = missingAcct.Quote(ctx, price.Money{Amount: decimal.NewFromInt(10), Currency: currency.EUR})
	require.Error(t, err)
	_, err = missingAcct.Provider(ctx)
	require.Error(t, err)
}

func testProviderSync(t *testing.T) {
	previous := registry
	registry = &providerRegistry{backends: orderedmap.NewOrderedMap[string, ProviderBackend]()}
	t.Cleanup(func() { registry = previous })

	ctx := t.Context()
	_, err := queries.GetQuerySet(&Payment{}).WithContext(ctx).Delete()
	require.NoError(t, err)
	qs := ProvidersQuery().WithContext(ctx)
	_, err = qs.Delete()
	require.NoError(t, err)

	_, err = qs.BulkCreate([]*ProviderAccount{
		{InternalName: "disabled_existing_psp", Enabled: true},
		{InternalName: "enabled_existing_psp", Enabled: true},
		{InternalName: "retired_psp", Enabled: true},
	})
	require.NoError(t, err)

	_, err = qs.Filter("InternalName", "disabled_existing_psp").
		BulkUpdate(expr.As("Enabled", expr.V(false)))
	require.NoError(t, err)

	RegisterProvider(&PSP{Name: "disabled_existing_psp", CurrencyUnit: currency.EUR})
	RegisterProvider(&PSP{Name: "enabled_existing_psp", CurrencyUnit: currency.EUR})
	RegisterProvider(&PSP{Name: "new_registered_psp", CurrencyUnit: currency.EUR})

	for range 2 {
		require.NoError(t, qs.Sync())
		rows, err := qs.All()
		require.NoError(t, err)

		got := make(map[string]bool, len(rows))
		for _, row := range rows {
			require.False(t, row.Object.ID.IsZero())
			got[row.Object.InternalName] = row.Object.Enabled
		}
		require.Equal(t, map[string]bool{
			"disabled_existing_psp": false,
			"enabled_existing_psp":  true,
			"retired_psp":           false,
			"new_registered_psp":    true,
		}, got)
	}

	// Syncing with no registered providers disables all rows.
	registry = &providerRegistry{backends: orderedmap.NewOrderedMap[string, ProviderBackend]()}
	require.NoError(t, qs.Sync())
	rows, err := qs.All()
	require.NoError(t, err)
	for _, row := range rows {
		require.False(t, row.Object.Enabled, "expected %s to be disabled when registry is empty", row.Object.InternalName)
	}
}
