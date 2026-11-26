package payments

import (
	"context"
	"database/sql"
	"time"

	"github.com/Nigel2392/go-django/contrib/shop/shared/relations/modelkeys"
	"github.com/Nigel2392/go-django/contrib/shop/util/paystate"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/shopspring/decimal"
)

type Payment struct {
	ID               drivers.ULID
	Order            attrs.Definer
	Account          *ProviderAccount
	Provider         string
	TransactionID    sql.NullString
	CheckoutURL      string
	Amount           decimal.Decimal
	Fee              decimal.Decimal
	Currency         string
	State            paystate.PayState
	CreatedAt        drivers.Timestamp
	UpdatedAt        drivers.Timestamp
	AuthorizationRef string
	CaptureRef       string
	OperationKey     string
	OperationVersion uint64
	Operations       drivers.JSON[map[string]OperationRecord]
	History          drivers.JSON[[]PaymentEvent]
}

// PaymentEvent is a recorded step in a payment attempt, in write order.
type PaymentEvent struct {
	At        time.Time
	Kind      PaymentEventKind
	State     paystate.PayState
	RequestID string         `json:",omitempty"`
	Operation OperationKind  `json:",omitempty"`
	Phase     OperationState `json:",omitempty"`
	Amount    *Money         `json:",omitempty"`
}

type PaymentEventKind string

const (
	EventPrepared       PaymentEventKind = "prepared"
	EventCheckoutOpened PaymentEventKind = "checkout_opened"
	EventCheckoutFailed PaymentEventKind = "checkout_failed"
	EventStatusUpdated  PaymentEventKind = "status_updated"
	EventOperation      PaymentEventKind = "operation"
)

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
		paymentField(m, "Currency", &m.Currency, attrs.Config{MaxLength: 3, Blank: true, Default: ""}),
		paymentField(m, "CreatedAt", &m.CreatedAt, attrs.Config{AutoNowAdd: true}),
		paymentField(m, "UpdatedAt", &m.UpdatedAt, attrs.Config{AutoNow: true}),
		paymentField(m, "AuthorizationRef", &m.AuthorizationRef, attrs.Config{MaxLength: 255, Blank: true, Default: ""}),
		paymentField(m, "CaptureRef", &m.CaptureRef, attrs.Config{MaxLength: 255, Blank: true, Default: ""}),
		paymentField(m, "OperationKey", &m.OperationKey, attrs.Config{MaxLength: 128, Blank: true, Default: ""}),
		paymentField(m, "OperationVersion", &m.OperationVersion, attrs.Config{Null: true, Blank: true, Default: uint64(0)}),
		paymentField(m, "Operations", &m.Operations, attrs.Config{Null: true, Blank: true}),
		paymentField(m, "History", &m.History, attrs.Config{Null: true, Blank: true, ReadOnly: true}),
	)
}
