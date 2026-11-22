package public

import (
	"net/http"

	"github.com/Nigel2392/go-django/contrib/shop/payments"
	"github.com/Nigel2392/go-django/contrib/shop/shared/api"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/logger"
	"golang.org/x/text/currency"
)

type PaymentOption struct {
	AccountID drivers.ULID `json:"account_id"`
	Label     string       `json:"label"`
	payments.Quote
}

// PaymentOptions quotes enabled accounts in the shop currency for the active cart.
func (c CartAPI) PaymentOptions(w http.ResponseWriter, r *http.Request) {
	unit, err := currency.ParseISO(c.Currency)
	if err != nil || unit.String() != c.Currency {
		api.RespondError(w, http.StatusServiceUnavailable, "shop payment currency is unavailable")
		return
	}
	ctx := r.Context()
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
	accounts, err := queries.
		GetQuerySet(&payments.ProviderAccount{}).
		WithContext(ctx).
		Filter("Enabled", true).
		Filter("Currency", c.Currency).
		OrderBy("Label", "ID").
		All()
	if err != nil {
		logger.Warnf("error loading accounts for payment options: %v", err)
		api.RespondError(w, http.StatusInternalServerError, "failed to load payment options")
		return
	}
	subtotal := payments.Money{
		Amount:   cart.Totals.Payable(),
		Currency: c.Currency,
	}
	options := make([]PaymentOption, 0, len(accounts))
	for _, row := range accounts {
		account := row.Object
		if !payments.ProviderRegistered(account.Driver) {
			continue
		}
		quote, err := account.Quote(subtotal)
		if err != nil {
			logger.Warnf("error quoting payment account %s: %v", account.ID, err)
			continue
		}
		options = append(options, PaymentOption{AccountID: account.ID, Label: account.Label, Quote: quote})
	}
	if err := api.Respond(w, options); err != nil {
		logger.Warnf("error writing payment options: %v", err)
	}
}
