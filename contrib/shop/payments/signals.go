package payments

import (
	"github.com/Nigel2392/go-django/contrib/shop/util/signals"
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

type productSignalManager struct {
	Started  signals.Signal[PaymentStartedSignal]
	Updated  signals.Signal[PaymentUpdatedSignal]
	Refunded signals.Signal[PaymentRefundedSignal]
}

var Signals = productSignalManager{
	Started:  signals.Pool.Get[PaymentStartedSignal]("shop.payments.Payment.Started"),
	Updated:  signals.Pool.Get[PaymentUpdatedSignal]("shop.payments.Payment.Updated"),
	Refunded: signals.Pool.Get[PaymentRefundedSignal]("shop.payments.Payment.Refunded"),
}
