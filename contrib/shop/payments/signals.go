package payments

import (
	"github.com/Nigel2392/go-django/contrib/shop/shared/signals"
)

type PaymentStartedSignal struct {
	signals.BaseSignal
}

type PaymentUpdatedSignal struct {
	signals.BaseSignal
}

type PaymentRefundedSignal struct {
	signals.BaseSignal
}

type PaymentVoidedSignal struct {
	signals.BaseSignal
}

type productSignalManager struct {
	Start  signals.Signal[PaymentStartedSignal]
	Update signals.Signal[PaymentUpdatedSignal]
	Refund signals.Signal[PaymentRefundedSignal]
	Void   signals.Signal[PaymentVoidedSignal]
}

var Signals = productSignalManager{
	Start:  signals.Pool.Get[PaymentStartedSignal]("shop.payments.Payment.Start"),
	Update: signals.Pool.Get[PaymentUpdatedSignal]("shop.payments.Payment.Update"),
	Refund: signals.Pool.Get[PaymentRefundedSignal]("shop.payments.Payment.Refund"),
	Void:   signals.Pool.Get[PaymentVoidedSignal]("shop.payments.Payment.Void"),
}
