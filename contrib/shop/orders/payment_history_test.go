package orders

import (
	"strings"
	"testing"
	"time"

	"github.com/Nigel2392/go-django/contrib/shop/payments"
	"github.com/Nigel2392/go-django/contrib/shop/shared/price"
	"github.com/Nigel2392/go-django/contrib/shop/util/paystate"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/shopspring/decimal"
	"golang.org/x/text/currency"
)

func TestPaymentHistoryHTML(t *testing.T) {
	payment := &payments.Payment{
		Provider:       "<unsafe>",
		State:          paystate.STATUS_PARTIAL_REFUND,
		Currency:       "EUR",
		Amount:         decimal.NewFromInt(25),
		Fee:            decimal.RequireFromString("0.50"),
		RefundedAmount: decimal.NewFromInt(5),
		ProviderFee: drivers.JSON[*price.Money]{
			Data: &price.Money{
				Amount:   decimal.RequireFromString("0.28"),
				Currency: currency.USD,
			},
		},
		History: drivers.JSON[[]payments.PaymentEvent]{
			Data: []payments.PaymentEvent{
				{
					At:    time.Unix(0, 0),
					Kind:  payments.EventCheckoutOpened,
					State: paystate.STATUS_PENDING,
				},
				{
					At:        time.Unix(1, 0),
					Kind:      payments.EventRefunded,
					State:     paystate.STATUS_PARTIAL_REFUND,
					RequestID: "refund-1",
					Amount: &price.Money{
						Amount:   decimal.NewFromInt(5),
						Currency: currency.EUR,
					},
				},
			},
		},
	}
	html := string(paymentHistoryHTML(t.Context(), payment))
	for _, want := range []string{
		"1970-01-01T00:00:00Z",
		"Provider checkout opened",
		"refund-1",
		"&lt;unsafe&gt;",
		"Subtotal before payment fee: 24.5 EUR",
		"Customer payment fee: 0.5 EUR",
		"Provider processing fee: 0.28 USD",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %q from %s", want, html)
		}
	}
	if strings.Count(html, "refund-1") != 1 || strings.Count(html, "<time ") != 2 || strings.Contains(html, "<unsafe>") {
		t.Fatalf("unexpected timeline markup: %s", html)
	}
}
