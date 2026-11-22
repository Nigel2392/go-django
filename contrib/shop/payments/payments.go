package payments

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"github.com/Nigel2392/go-django/contrib/shop/util/paystate"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/shopspring/decimal"
)

type Money struct {
	Amount   decimal.Decimal
	Currency string
}

type CreatePaymentRequest struct {
	AttemptID  drivers.ULID
	Amount     Money
	ReturnURL  string
	WebhookURL string
}

type PaymentStatus struct {
	ProviderRef string
	State       paystate.PayState
	// Amount is the confirmed order-currency amount when paid and the original
	// order-currency amount in other states.
	Amount Money
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

var providers sync.Map

func RegisterProvider(key string, factory ProviderFactory) {
	if key == "" || factory == nil {
		panic("payments: provider key and factory are required")
	}
	if _, loaded := providers.LoadOrStore(key, factory); loaded {
		panic("payments: duplicate provider " + key)
	}
}

func (a *ProviderAccount) provider(ctx context.Context) (Provider, error) {
	factory, ok := providers.Load(a.Driver)
	if !ok {
		return nil, fmt.Errorf("payment provider %q is unavailable", a.Driver)
	}
	return factory.(ProviderFactory)(ctx, a)
}

func ProviderRegistered(key string) bool {
	_, ok := providers.Load(key)
	return ok
}
