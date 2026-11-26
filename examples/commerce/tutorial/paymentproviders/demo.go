package paymentproviders

import (
	"context"
	"fmt"
	"sync"

	"github.com/Nigel2392/go-django/contrib/shop/payments"
	"github.com/Nigel2392/go-django/contrib/shop/util/paystate"
)

const DemoDriver = "tutorial"

var _ payments.Provider = (*Demo)(nil)

// Demo simulates a merchant's provider in memory for the checkout tutorial.
type Demo struct {
	mu        sync.Mutex
	checkouts map[string]payments.Checkout
}

func RegisterDemo() *Demo {
	demo := &Demo{checkouts: make(map[string]payments.Checkout)}
	payments.RegisterProvider(DemoDriver, func(context.Context, *payments.ProviderAccount) (payments.Provider, error) {
		return demo, nil
	})
	return demo
}

func (d *Demo) CreatePayment(_ context.Context, request payments.CreatePaymentRequest) (payments.Checkout, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	ref := request.AttemptID.String()
	if checkout, exists := d.checkouts[ref]; exists {
		return checkout, nil
	}
	checkout := payments.Checkout{
		PaymentStatus: payments.PaymentStatus{
			ProviderRef: ref,
			State:       paystate.STATUS_PENDING,
			Amount:      request.Amount,
		},
		URL: "https://checkout.example.test/pay/" + ref,
	}
	d.checkouts[ref] = checkout
	return checkout, nil
}

func (d *Demo) GetPayment(_ context.Context, ref string) (payments.PaymentStatus, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	checkout, exists := d.checkouts[ref]
	if !exists {
		return payments.PaymentStatus{}, fmt.Errorf("unknown demo payment %q", ref)
	}
	return checkout.PaymentStatus, nil
}

func (d *Demo) HandleWebhook(context.Context, payments.WebhookRequest) (payments.WebhookResult, error) {
	return payments.WebhookResult{}, payments.ErrInvalidWebhook
}

// MarkPaid simulates a provider-side payment that Refresh can reconcile.
func (d *Demo) MarkPaid(ref string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	checkout, exists := d.checkouts[ref]
	if !exists {
		return fmt.Errorf("unknown demo payment %q", ref)
	}
	checkout.State = paystate.STATUS_PAID
	d.checkouts[ref] = checkout
	return nil
}
