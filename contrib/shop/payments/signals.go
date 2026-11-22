package payments

import (
	"github.com/Nigel2392/go-django/contrib/shop/shared/signals"
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
	Amount    Money
	Result    OperationResult
}

type PaymentVoidedSignal struct {
	signals.BaseSignal
	Payment   *Payment
	RequestID string
}

type PaymentOperationSignal struct {
	signals.BaseSignal
	Payment   *Payment
	RequestID string
	Operation OperationRecord
	Error     error
}

type productSignalManager struct {
	Start              signals.Signal[PaymentStartedSignal]
	Update             signals.Signal[PaymentUpdatedSignal]
	Refund             signals.Signal[PaymentRefundedSignal]
	Void               signals.Signal[PaymentVoidedSignal]
	Capture            signals.Signal[PaymentOperationSignal]
	Cancel             signals.Signal[PaymentOperationSignal]
	OperationRequested signals.Signal[PaymentOperationSignal]
	OperationPending   signals.Signal[PaymentOperationSignal]
	OperationFailed    signals.Signal[PaymentOperationSignal]
}

var Signals = productSignalManager{
	Start:              signals.Pool.Get[PaymentStartedSignal]("shop.payments.Payment.Start"),
	Update:             signals.Pool.Get[PaymentUpdatedSignal]("shop.payments.Payment.Update"),
	Refund:             signals.Pool.Get[PaymentRefundedSignal]("shop.payments.Payment.Refund"),
	Void:               signals.Pool.Get[PaymentVoidedSignal]("shop.payments.Payment.Void"),
	Capture:            signals.Pool.Get[PaymentOperationSignal]("shop.payments.Payment.Capture"),
	Cancel:             signals.Pool.Get[PaymentOperationSignal]("shop.payments.Payment.Cancel"),
	OperationRequested: signals.Pool.Get[PaymentOperationSignal]("shop.payments.Payment.OperationRequested"),
	OperationPending:   signals.Pool.Get[PaymentOperationSignal]("shop.payments.Payment.OperationPending"),
	OperationFailed:    signals.Pool.Get[PaymentOperationSignal]("shop.payments.Payment.OperationFailed"),
}
