package handlers

import (
	"context"

	"github.com/Nigel2392/go-django/contrib/shop/payments"
	"github.com/Nigel2392/go-django/contrib/shop/util/paystate"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/logger"
)

func registerPaymentAudits() {
	listenAudit(payments.Signals.Start, "shop:payments:start", paymentStartedDefinition{}, func(s payments.PaymentStartedSignal) auditEvent {
		return paymentAudit(s.Payment, paymentStateLevel(s.Payment))
	})
	listenAudit(payments.Signals.Update, "shop:payments:update", paymentUpdatedDefinition{}, func(s payments.PaymentUpdatedSignal) auditEvent {
		entry := paymentAudit(s.Payment, paymentStateLevel(s.Payment))
		entry.data["previous_state"] = s.PreviousState
		return entry
	})
	listenAudit(payments.Signals.Refund, "shop:payments:refund", paymentRefundedDefinition{}, func(s payments.PaymentRefundedSignal) auditEvent {
		entry := paymentAudit(s.Payment, logger.INF)
		entry.data["request_id"] = s.RequestID
		entry.data["refund_amount"] = s.Amount.Amount.String()
		entry.data["refund_currency"] = s.Amount.Currency.String()
		return entry
	})
	listenAudit(payments.Signals.Cancel, "shop:payments:cancel", paymentCancelledDefinition{}, func(s payments.PaymentCancelledSignal) auditEvent {
		entry := paymentAudit(s.Payment, logger.INF)
		entry.data["request_id"] = s.RequestID
		return entry
	})
}

func paymentStateLevel(payment *payments.Payment) logger.LogLevel {
	if payment != nil {
		switch payment.State {
		case paystate.STATUS_FAILURE:
			return logger.ERR
		case paystate.STATUS_DENIED, paystate.STATUS_VERIFY, paystate.STATUS_PAID_CHECKAMOUNT, paystate.STATUS_CHARGEBACK:
			return logger.WRN
		}
	}
	return logger.INF
}

func paymentAudit(payment *payments.Payment, level logger.LogLevel) auditEvent {
	data := make(map[string]any)
	if payment != nil {
		data["state"] = payment.State
		data["amount"] = payment.Amount.String()
		data["currency"] = payment.Currency
		data["provider"] = payment.Provider
		if payment.Account != nil {
			data["account_id"] = payment.Account.ID
		}
		if !attrs.IsZero(payment.Order) {
			data["order_id"] = attrs.Define(context.Background(), payment.Order).Primary().GetValue()
		}
	}
	return auditEvent{object: payment, level: level, data: data}
}
