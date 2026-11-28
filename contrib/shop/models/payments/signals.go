package payments

import (
	"github.com/Nigel2392/go-django/contrib/shop/core/shared/price"
	"github.com/Nigel2392/go-django/contrib/shop/core/shared/signals"
)

type PaymentStartedSignal struct {
	signals.BaseSignal
	Payment *Payment
}

type PaymentUpdatedSignal struct {
	signals.BaseSignal
	Payment       *Payment
	PreviousState string
}

type PaymentRefundedSignal struct {
	signals.BaseSignal
	Payment   *Payment
	RequestID string
	Amount    price.Money
	Result    RefundResult
}

type PaymentCancelledSignal struct {
	signals.BaseSignal
	Payment   *Payment
	RequestID string
}

type paymentSignalManager struct {
	Start  signals.Signal[PaymentStartedSignal]
	Update signals.Signal[PaymentUpdatedSignal]
	Refund signals.Signal[PaymentRefundedSignal]
	Cancel signals.Signal[PaymentCancelledSignal]
}

var Signals = paymentSignalManager{
	Start:  signals.Pool.Get[PaymentStartedSignal]("shop.payments.Payment.Start"),
	Update: signals.Pool.Get[PaymentUpdatedSignal]("shop.payments.Payment.Update"),
	Refund: signals.Pool.Get[PaymentRefundedSignal]("shop.payments.Payment.Refund"),
	Cancel: signals.Pool.Get[PaymentCancelledSignal]("shop.payments.Payment.Cancel"),
}
