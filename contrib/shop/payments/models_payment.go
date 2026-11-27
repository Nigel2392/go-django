package payments

import (
	"context"
	"database/sql"
	"time"

	"github.com/Nigel2392/go-django/contrib/shop/shared/price"
	"github.com/Nigel2392/go-django/contrib/shop/shared/relations/modelkeys"
	"github.com/Nigel2392/go-django/contrib/shop/util/paystate"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/shopspring/decimal"
	"golang.org/x/text/currency"
)

type Payment struct {
	ID            drivers.ULID
	Order         attrs.Definer
	Account       *ProviderAccount
	Provider      string
	TransactionID sql.NullString
	CheckoutURL   string
	State         paystate.PayState
	Amount        decimal.Decimal

	// Fee is the customer-facing payment fee included in Amount.
	Fee decimal.Decimal

	// RefundedAmount is the cumulative refunded amount in the payment's Currency.
	RefundedAmount decimal.Decimal

	// ProviderFee is the latest merchant processing fee reported by the PSP.
	ProviderFee drivers.JSON[*price.Money]
	Currency    string
	CreatedAt   drivers.Timestamp
	UpdatedAt   drivers.Timestamp
	History     drivers.JSON[[]PaymentEvent]
}

// CurrencyUnit parses the snapshotted ISO currency code stored on the Payment.
func (m *Payment) CurrencyUnit() currency.Unit {
	if m == nil || m.Currency == "" {
		return currency.Unit{}
	}
	unit, err := currency.ParseISO(m.Currency)
	if err != nil || unit.String() != m.Currency {
		return currency.Unit{}
	}
	return unit
}

// PaymentEvent is a recorded step in a payment attempt, in write order.
type PaymentEvent struct {
	At          time.Time
	Kind        PaymentEventKind
	State       paystate.PayState
	RequestID   string       `json:",omitempty"`
	ProviderRef string       `json:",omitempty"`
	Amount      *price.Money `json:",omitempty"`
	ProviderFee *price.Money `json:",omitempty"`
}

type PaymentEventKind string

const (
	EventCheckoutOpened     PaymentEventKind = "checkout_opened"
	EventStatusUpdated      PaymentEventKind = "status_updated"
	EventProviderFeeUpdated PaymentEventKind = "provider_fee_updated"
	EventNoPaymentRequired  PaymentEventKind = "no_payment_required"
	EventRefunded           PaymentEventKind = "refunded"
	EventCancelled          PaymentEventKind = "cancelled"
)

const freeCheckoutProvider = "shop:free-checkout"

// IsFreeCheckout reports whether this is the internal record for a zero-total order.
func (m *Payment) IsFreeCheckout() bool {
	return m.Provider == freeCheckoutProvider &&
		m.State == paystate.STATUS_NO_PAYMENT_REQUIRED &&
		(m.Account == nil || m.Account.ID.IsZero()) &&
		!m.TransactionID.Valid &&
		m.Amount.IsZero() &&
		m.Fee.IsZero()
}

func (m *Payment) BeforeCreate(context.Context) error {
	if m.ID.IsZero() {
		m.ID = drivers.NewULID()
	}
	return nil
}

func (m *Payment) DatabaseIndexes(attrs.Definer) []migrator.Index {
	return []migrator.Index{
		{Identifier: "shop_payment_account_ref_idx", Fields: []string{"Account", "TransactionID"}, Unique: true},
	}
}

func (m *Payment) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		paymentField(m, "ID", &m.ID, attrs.Config{Primary: true, ReadOnly: true}),
		paymentField(m, "Order", &m.Order, attrs.Config{
			Column: "order_id", Null: true, Blank: true, ReverseAlias: "Payments",
			RelForeignKey: attrs.RelatedDeferred(attrs.RelManyToOne, modelkeys.OrderModelKey, "", nil),
		}),
		paymentField(m, "Account", &m.Account, attrs.Config{
			Column: "account_id", Null: true, Blank: true,
			RelForeignKey: attrs.Relate(&ProviderAccount{}, "", nil),
		}),
		paymentField(m, "Provider", &m.Provider, attrs.Config{MaxLength: 75}),
		paymentField(m, "TransactionID", &m.TransactionID, attrs.Config{MaxLength: 255, Null: true, Blank: true}),
		paymentField(m, "CheckoutURL", &m.CheckoutURL, attrs.Config{MaxLength: 4096, Blank: true, Default: ""}),
		paymentField(m, "State", &m.State, attrs.Config{MaxLength: 40}),
		paymentField(m, "Amount", &m.Amount, attrs.Config{Precision: 13, Scale: 4}),
		paymentField(m, "Fee", &m.Fee, attrs.Config{Precision: 13, Scale: 4, Blank: true, Default: decimal.Zero}),
		paymentField(m, "RefundedAmount", &m.RefundedAmount, attrs.Config{Precision: 13, Scale: 4, Blank: true, Default: decimal.Zero}),
		paymentField(m, "ProviderFee", &m.ProviderFee, attrs.Config{Null: true, Blank: true, ReadOnly: true}),
		paymentField(m, "Currency", &m.Currency, attrs.Config{MaxLength: 3, Blank: true, Default: ""}),
		paymentField(m, "CreatedAt", &m.CreatedAt, attrs.Config{AutoNowAdd: true}),
		paymentField(m, "UpdatedAt", &m.UpdatedAt, attrs.Config{AutoNow: true}),
		paymentField(m, "History", &m.History, attrs.Config{Null: true, Blank: true, ReadOnly: true}),
	)
}
