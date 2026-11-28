package public

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	"github.com/Nigel2392/go-django/contrib/shop/core/shared/api"
	"github.com/Nigel2392/go-django/contrib/shop/core/shared/price"
	"github.com/Nigel2392/go-django/contrib/shop/models/payments"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/shopspring/decimal"
	"golang.org/x/text/currency"
)

type PaymentOption struct {
	AccountID         drivers.ULID `json:"account_id"`
	Label             string       `json:"label"`
	NoPaymentRequired bool         `json:"no_payment_required,omitempty"`
	price.Quote
}

type paymentOptionJSON struct {
	AccountID         drivers.ULID    `json:"account_id"`
	Label             string          `json:"label"`
	NoPaymentRequired bool            `json:"no_payment_required,omitempty"`
	Subtotal          decimal.Decimal `json:"subtotal"`
	Fee               decimal.Decimal `json:"fee"`
	Total             decimal.Decimal `json:"total"`
	Currency          string          `json:"currency"`
}

func (o PaymentOption) MarshalJSON() ([]byte, error) {
	var code string
	if o.Currency != (currency.Unit{}) {
		code = o.Currency.String()
	}
	return json.Marshal(paymentOptionJSON{
		AccountID:         o.AccountID,
		Label:             o.Label,
		NoPaymentRequired: o.NoPaymentRequired,
		Subtotal:          o.Subtotal,
		Fee:               o.Fee,
		Total:             o.Total,
		Currency:          code,
	})
}

func (o *PaymentOption) UnmarshalJSON(data []byte) error {
	var raw paymentOptionJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	var unit currency.Unit
	if raw.Currency != "" {
		parsed, err := currency.ParseISO(raw.Currency)
		if err != nil || parsed.String() != raw.Currency {
			return fmt.Errorf("invalid ISO currency %q", raw.Currency)
		}
		unit = parsed
	}
	o.AccountID = raw.AccountID
	o.Label = raw.Label
	o.NoPaymentRequired = raw.NoPaymentRequired
	o.Quote = price.Quote{
		Subtotal: raw.Subtotal,
		Fee:      raw.Fee,
		Total:    raw.Total,
		Currency: unit,
	}
	return nil
}

type PaymentsAPI struct {
	Currency    currency.Unit
	GetCurrency func(context.Context) currency.Unit
}

func (c PaymentsAPI) resolveCurrency(ctx context.Context) currency.Unit {
	if c.GetCurrency != nil {
		if unit := c.GetCurrency(ctx); unit != (currency.Unit{}) {
			return unit
		}
	}
	return c.Currency
}

// PaymentOptions quotes checkout in the shop currency for the active cart.
func (c PaymentsAPI) PaymentOptions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	unit := c.resolveCurrency(ctx)
	if unit == (currency.Unit{}) {
		api.RespondError(w, http.StatusServiceUnavailable, "shop payment currency is unavailable")
		return
	}

	cart, err := cartQuery(ctx, r, false).
		CalcOne()
	if errors.Is(err, errors.NoRows) {
		api.RespondError(w, http.StatusNotFound, "cart not found")
		return
	}
	if err != nil {
		logger.Warnf("error calculating cart for payment options: %v", err)
		api.RespondError(w, http.StatusInternalServerError, "failed to calculate payment options")
		return
	}

	subtotal := price.Money{
		Amount:   cart.Totals.Payable(),
		Currency: unit,
	}
	quote, err := price.QuoteFee(subtotal, decimal.Zero, decimal.Zero)
	if len(cart.Items) > 0 && err == nil && quote.Total.IsZero() {
		options := []PaymentOption{
			{
				Label:             trans.T(ctx, "No payment required"),
				NoPaymentRequired: true,
				Quote:             quote,
			},
		}
		if err := api.Respond(w, options); err != nil {
			logger.Warnf("error writing payment options: %v", err)
		}
		return
	}

	accounts, err := queries.
		GetQuerySet(&payments.ProviderAccount{}).
		WithContext(ctx).
		Filter("Enabled", true).
		All()
	if err != nil {
		logger.Warnf("error loading accounts for payment options: %v", err)
		api.RespondError(w, http.StatusInternalServerError, "failed to load payment options")
		return
	}

	type optionEntry struct {
		ordering int
		label    string
		id       string
		option   PaymentOption
	}
	entries := make([]optionEntry, 0, len(accounts))
	for _, row := range accounts {
		account := row.Object
		backend, ok := payments.RetrieveProvider(account.InternalName)
		if !ok || backend.Currency(ctx) != unit {
			continue
		}
		quote, err := account.Quote(ctx, subtotal)
		if err != nil {
			logger.Warnf("error quoting payment account %s: %v", account.ID, err)
			continue
		}
		label := backend.Label(ctx)
		entries = append(entries, optionEntry{
			ordering: backend.Ordering(),
			label:    label,
			id:       account.ID.String(),
			option: PaymentOption{
				AccountID: account.ID,
				Label:     label,
				Quote:     quote,
			},
		})
	}
	slices.SortFunc(entries, func(a, b optionEntry) int {
		return cmp.Or(
			cmp.Compare(a.ordering, b.ordering),
			cmp.Compare(a.label, b.label),
			cmp.Compare(a.id, b.id),
		)
	})
	options := make([]PaymentOption, len(entries))
	for i, entry := range entries {
		options[i] = entry.option
	}
	if err := api.Respond(w, options); err != nil {
		logger.Warnf("error writing payment options: %v", err)
	}
}
