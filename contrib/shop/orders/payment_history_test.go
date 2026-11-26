package orders

import (
	"strings"
	"testing"
	"time"

	"github.com/Nigel2392/go-django/contrib/shop/payments"
	"github.com/Nigel2392/go-django/contrib/shop/util/paystate"
	"github.com/Nigel2392/go-django/queries/src/drivers"
)

func TestPaymentHistoryHTML(t *testing.T) {
	payment := &payments.Payment{
		Provider: "<unsafe>",
		State:    paystate.STATUS_PAID,
		History: drivers.JSON[[]payments.PaymentEvent]{
			Data: []payments.PaymentEvent{
				{
					At:    time.Unix(0, 0),
					Kind:  payments.EventPrepared,
					State: paystate.STATUS_INIT,
				},
				{
					At:        time.Unix(1, 0),
					Kind:      payments.EventOperation,
					Operation: payments.OperationCapture,
					Phase:     payments.OperationSucceeded,
					State:     paystate.STATUS_PAID,
					RequestID: "capture-new",
				},
			},
		},
		Operations: drivers.JSON[map[string]payments.OperationRecord]{
			Data: map[string]payments.OperationRecord{
				"capture-new": {},
				"refund-old": {
					Kind:  payments.OperationRefund,
					Phase: payments.OperationFailed,
				},
			},
		},
	}
	html := string(paymentHistoryHTML(t.Context(), payment))
	for _, want := range []string{"1970-01-01T00:00:00Z", "Payment prepared", "Capture · Succeeded", "Refund · Failed", "&lt;unsafe&gt;"} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %q from %s", want, html)
		}
	}
	if strings.Count(html, "capture-new") != 1 || strings.Count(html, "<time ") != 2 || strings.Contains(html, "<unsafe>") {
		t.Fatalf("unexpected timeline markup: %s", html)
	}
}
