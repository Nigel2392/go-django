package payments

import (
	"context"
	"net/http"
	"net/url"

	"github.com/Nigel2392/go-django/contrib/shop/core/shared/price"
	"github.com/Nigel2392/go-django/contrib/shop/core/util/paystate"
	"github.com/Nigel2392/go-django/queries/src/drivers"
)

type CreatePaymentRequest struct {
	AttemptID  drivers.ULID
	Amount     price.Money
	ReturnURL  string
	WebhookURL string
}

type PaymentStatus struct {
	ProviderRef string
	State       paystate.PayState
	// Amount is the confirmed order-currency amount when paid and the original
	// order-currency amount in other states.
	Amount price.Money
	// ProviderFee is the processing fee reported by the PSP in its own currency.
	// An omitted fee leaves any previously reported fee in place.
	ProviderFee *price.Money
}

type Checkout struct {
	PaymentStatus
	URL string
}

type WebhookRequest struct {
	Method string
	Header http.Header
	URL    *url.URL
	Body   []byte
}

type WebhookResult struct {
	Payment     PaymentStatus
	Acknowledge func(http.ResponseWriter)
}

// Provider covers checkout and reconciliation. Additional payment operations
// can be exposed as capabilities by the adapters that support them.
type Provider interface {
	// CreatePayment must use AttemptID to make retried requests idempotent.
	CreatePayment(context.Context, CreatePaymentRequest) (Checkout, error)
	GetPayment(context.Context, string) (PaymentStatus, error)
	// HandleWebhook must authenticate or independently verify provider data.
	HandleWebhook(context.Context, WebhookRequest) (WebhookResult, error)
}

type ProviderFactory func(context.Context, *ProviderAccount) (Provider, error)

type RefundRequest struct {
	ProviderRef string
	RequestID   string
	Amount      price.Money
}

type RefundResult struct {
	ProviderRef string
}

type Refunder interface {
	RefundPayment(context.Context, RefundRequest) (RefundResult, error)
}

type CancelRequest struct {
	ProviderRef string
	RequestID   string
}

type Canceller interface {
	CancelPayment(context.Context, CancelRequest) error
}
