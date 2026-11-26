package payments

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"golang.org/x/text/currency"

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
	// Resource references used by optional capture, void and refund operations.
	AuthorizationRef string
	CaptureRef       string
	State            paystate.PayState
	// Amount is the confirmed order-currency amount when paid and the original
	// order-currency amount in other states.
	Amount Money
	// ProviderFee is the processing fee reported by the PSP in its own currency.
	// An omitted fee leaves any previously reported fee in place.
	ProviderFee *Money
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

// OperationRequest identifies a provider resource and merchant operation.
// RequestID stays stable on retries and changes for each distinct operation.
// The adapter selects the resource reference for each supported operation.
type OperationRequest struct {
	ProviderRef string
	RequestID   string
}

func (r OperationRequest) Validate() error {
	if strings.TrimSpace(r.ProviderRef) == "" || strings.TrimSpace(r.RequestID) == "" {
		return errors.New("provider reference and operation request ID are required")
	}

	return nil
}

type RefundRequest struct {
	OperationRequest
	Amount Money
}

func (r RefundRequest) Validate() error {
	if err := r.OperationRequest.Validate(); err != nil {
		return err
	}

	unit, err := currency.ParseISO(r.Amount.Currency)
	if err != nil || unit.String() != r.Amount.Currency || !r.Amount.Amount.IsPositive() {
		return errors.New("refund requires a positive amount and ISO currency")
	}

	scale, _ := currency.Standard.Rounding(unit)
	if !r.Amount.Amount.Equal(r.Amount.Amount.Round(int32(scale))) {
		return errors.New("refund amount exceeds currency precision")
	}

	return nil
}

// OperationResult contains the operation reference and native provider status.
// Acceptance can precede settlement. Reconcile pending operations before
// changing accounting balances or retrying a provider without idempotency.
type OperationResult struct {
	ProviderRef string
	Status      string
	// An empty state means the operation is awaiting reconciliation.
	State OperationState
}

type OperationState string

const (
	OperationRequested OperationState = "requested"
	OperationUncertain OperationState = "uncertain"
	OperationPending   OperationState = "pending"
	OperationSucceeded OperationState = "succeeded"
	OperationFailed    OperationState = "failed"
)

type OperationKind string

const (
	OperationRefund  OperationKind = "refund"
	OperationVoid    OperationKind = "void"
	OperationCapture OperationKind = "capture"
	OperationCancel  OperationKind = "cancel"
)

const (
	OperationStatusVoided    = "VOIDED"
	OperationStatusCancelled = "CANCELLED"
)

type Voider interface {
	VoidPayment(context.Context, OperationRequest) error
}

type Refunder interface {
	RefundPayment(context.Context, RefundRequest) (OperationResult, error)
}

// Capturer captures the remaining authorized amount.
type Capturer interface {
	CapturePayment(context.Context, OperationRequest) (OperationResult, error)
}

type Canceller interface {
	CancelPayment(context.Context, OperationRequest) error
}

// OperationReferences identifies the resources used by each optional capability.
// Adapters with distinct authorization or capture resources implement
// OperationReferenceResolver. Other adapters use the payment reference.
type OperationReferences struct {
	Refund  string
	Void    string
	Capture string
	Cancel  string
}

type OperationReferenceResolver interface {
	OperationReferences(PaymentStatus) OperationReferences
}

// Provider resolves this account's adapter, including optional capabilities.
func (a *ProviderAccount) Provider(ctx context.Context) (Provider, error) {
	return a.provider(ctx)
}

var providers sync.Map

func RegisterProvider(key string, factory ProviderFactory) {
	if key == "" || factory == nil {
		panic("payments: provider key and factory are required")
	}
	if key == freeCheckoutProvider {
		panic("payments: reserved provider key " + key)
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
