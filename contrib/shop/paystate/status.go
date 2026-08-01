package paystate

import (
	"context"

	"github.com/Nigel2392/go-django/internal/strich"
	"github.com/Nigel2392/go-django/src/core/trans"
)

type PayState = string

var PAYSTATE strich.Check[PayState] = strich.Checker[PayState]{
	Delimiter: ':',
	Display: strich.CheckerDisplay[PayState]{
		Delimiter: " | ",
		Labels:    _statusTranslations,
	},
}

const (
	// Transaction is started (Paylink) no payment method is selected yet.
	STATUS_INIT PayState = "STATUS_INIT"

	// Transaction is created on the payment method, data must be provided by the Payer to proceed (bank selection / card data inserted)
	//
	// Data is received, send to the payment method, we wait for there response. Payment can be send into an processing status
	//
	// Data is provided to the Payer, we wait for the a synchronical response of the bank or third party. If we receive the Payment, transaction will be send into a processing state
	//
	// Provider or Issuer confirmed that the payment is send, but the funds are not received yet
	STATUS_PENDING PayState = "STATUS_PENDING"

	// The payment has been cancelled by the user
	STATUS_CANCEL PayState = "STATUS_CANCEL"

	// The payment or authorisation has expired.
	STATUS_EXPIRED PayState = "STATUS_EXPIRED"

	// The payment was rejected by the payment processor. In case of Buy now pay later methods this means that the customer did not pass the credit check.
	STATUS_DENIED PayState = "STATUS_DENIED"

	// The payment could not be processed due to an error at the payment method, processor or issuer.
	STATUS_FAILURE PayState = "STATUS_FAILURE"

	// Order is not completed, because the amount paid by the customer does not match the original order amount. If a user sends in a banktransfer that not equals the requested amount
	STATUS_PAID_CHECKAMOUNT PayState = "STATUS_PAID_CHECKAMOUNT"

	//	Partial payments are used with gift cards that do not cover the full amount of the order. The order can still change to PAID when a second transaction is performed.
	STATUS_PARTIAL_PAYMENT PayState = "STATUS_PARTIAL_PAYMENT"

	// The payment is treated as suspicious. The status needs to be determined by means of an additional check.
	STATUS_VERIFY PayState = "STATUS_VERIFY"

	// The payment has been reserved and can be captured. This status is used for credit card payments and BNPL.
	STATUS_AUTHORIZE PayState = "STATUS_AUTHORIZE"

	// The payment has been partly captured. You can void or capture the remaining autorised part.
	STATUS_PARTLY_CAPTURED PayState = "STATUS_PARTLY_CAPTURED"

	// The payment was successful.
	STATUS_PAID PayState = "STATUS_PAID"

	// Chargeback of a credit card payment.
	STATUS_CHARGEBACK PayState = "STATUS_CHARGEBACK"

	// The payment will be refunded (you can still cancel this).
	STATUS_REFUNDING PayState = "STATUS_REFUNDING"

	// The payment has been refunded.
	STATUS_REFUND PayState = "STATUS_REFUND"

	// The payment has been partially refunded.
	STATUS_PARTIAL_REFUND PayState = "STATUS_PARTIAL_REFUND"
)

var _statusTranslations = map[PayState]func(context.Context) string{
	STATUS_INIT:             trans.S("Initialised"),
	STATUS_PENDING:          trans.S("Pending"),
	STATUS_CANCEL:           trans.S("Cancelled"),
	STATUS_EXPIRED:          trans.S("Expired"),
	STATUS_DENIED:           trans.S("Denied"),
	STATUS_FAILURE:          trans.S("Failure"),
	STATUS_PAID_CHECKAMOUNT: trans.S("Check Amount"),
	STATUS_PARTIAL_PAYMENT:  trans.S("Partial Payment"),
	STATUS_VERIFY:           trans.S("Manual Verify"),
	STATUS_AUTHORIZE:        trans.S("Authorize"),
	STATUS_PARTLY_CAPTURED:  trans.S("Partly Captured"),
	STATUS_PAID:             trans.S("Paid"),
	STATUS_CHARGEBACK:       trans.S("Chargeback"),
	STATUS_REFUNDING:        trans.S("Refunding"),
	STATUS_REFUND:           trans.S("Refunded"),
	STATUS_PARTIAL_REFUND:   trans.S("Partial Refund"),
}
