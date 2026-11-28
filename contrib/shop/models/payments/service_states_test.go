//go:build test

package payments

import (
	"testing"

	"github.com/Nigel2392/go-django/contrib/shop/core/shared/price"
	"github.com/Nigel2392/go-django/contrib/shop/core/util/paystate"
	"github.com/shopspring/decimal"
	"golang.org/x/text/currency"
)

func TestPaymentStateValidation(t *testing.T) {
	p := &Payment{Currency: "EUR", Amount: decimal.NewFromInt(10)}
	for _, tc := range []struct {
		name   string
		status PaymentStatus
		valid  bool
	}{
		{"valid", PaymentStatus{ProviderRef: "ref", State: paystate.STATUS_PAID, Amount: price.Money{Amount: p.Amount, Currency: currency.EUR}}, true},
		{"blank state", PaymentStatus{ProviderRef: "ref", Amount: price.Money{Amount: p.Amount, Currency: currency.EUR}}, false},
		{"unrecognized state", PaymentStatus{ProviderRef: "ref", State: "COMPLETED", Amount: price.Money{Amount: p.Amount, Currency: currency.EUR}}, false},
		{"internal free checkout state", PaymentStatus{ProviderRef: "ref", State: paystate.STATUS_NO_PAYMENT_REQUIRED, Amount: price.Money{Amount: p.Amount, Currency: currency.EUR}}, false},
		{"blank reference", PaymentStatus{State: paystate.STATUS_PAID, Amount: price.Money{Amount: p.Amount, Currency: currency.EUR}}, false},
		{"fractional mismatch", PaymentStatus{ProviderRef: "ref", State: paystate.STATUS_PAID, Amount: price.Money{Amount: decimal.RequireFromString("10.001"), Currency: currency.EUR}}, false},
		{"currency mismatch", PaymentStatus{ProviderRef: "ref", State: paystate.STATUS_PAID, Amount: price.Money{Amount: p.Amount, Currency: currency.USD}}, false},
		{"missing currency", PaymentStatus{ProviderRef: "ref", State: paystate.STATUS_PAID, Amount: price.Money{Amount: p.Amount}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if (p.validateStatus(tc.status) == nil) != tc.valid {
				t.Fatal("incorrect validation")
			}
		})
	}
}

func TestTerminalPaymentStates(t *testing.T) {
	states := []string{paystate.STATUS_INIT, paystate.STATUS_PENDING, paystate.STATUS_CANCEL, paystate.STATUS_EXPIRED, paystate.STATUS_DENIED, paystate.STATUS_FAILURE, paystate.STATUS_PAID_CHECKAMOUNT, paystate.STATUS_PARTIAL_PAYMENT, paystate.STATUS_VERIFY, paystate.STATUS_AUTHORIZE, paystate.STATUS_PARTLY_CAPTURED, paystate.STATUS_PAID, paystate.STATUS_CHARGEBACK, paystate.STATUS_REFUNDING, paystate.STATUS_REFUND, paystate.STATUS_PARTIAL_REFUND}
	for _, from := range []string{paystate.STATUS_CANCEL, paystate.STATUS_EXPIRED, paystate.STATUS_DENIED, paystate.STATUS_FAILURE, paystate.STATUS_REFUND, paystate.STATUS_CHARGEBACK, paystate.STATUS_NO_PAYMENT_REQUIRED} {
		for _, to := range states {
			t.Run(from+"/"+to, func(t *testing.T) {
				if canTransition(from, to) {
					t.Fatal("terminal payment regressed")
				}
			})
		}
	}

	for _, to := range []string{paystate.STATUS_INIT, paystate.STATUS_PENDING, paystate.STATUS_AUTHORIZE, paystate.STATUS_FAILURE} {
		t.Run("paid/"+to, func(t *testing.T) {
			if canTransition(paystate.STATUS_PAID, to) {
				t.Fatal("settled payment regressed")
			}
		})
	}
}

func TestAllowedPaymentTransitions(t *testing.T) {
	for _, tc := range []struct {
		from, to string
		allowed  bool
	}{
		{paystate.STATUS_PENDING, paystate.STATUS_INIT, false},
		{paystate.STATUS_INIT, paystate.STATUS_NO_PAYMENT_REQUIRED, false},
		{paystate.STATUS_AUTHORIZE, paystate.STATUS_PENDING, false},
		{paystate.STATUS_VERIFY, paystate.STATUS_INIT, false},
		{paystate.STATUS_PARTIAL_PAYMENT, paystate.STATUS_PAID, true},
		{paystate.STATUS_AUTHORIZE, paystate.STATUS_PAID, true},
		{paystate.STATUS_PARTLY_CAPTURED, paystate.STATUS_PAID, true},
		{paystate.STATUS_PARTLY_CAPTURED, paystate.STATUS_VERIFY, true},
		{paystate.STATUS_PARTLY_CAPTURED, paystate.STATUS_PENDING, false},
		{paystate.STATUS_PAID, paystate.STATUS_REFUNDING, true},
		{paystate.STATUS_PAID, paystate.STATUS_CHARGEBACK, true},
		{paystate.STATUS_REFUNDING, paystate.STATUS_REFUND, true},
		{paystate.STATUS_PARTIAL_REFUND, paystate.STATUS_REFUND, true},
	} {
		t.Run(tc.from+"/"+tc.to, func(t *testing.T) {
			if canTransition(tc.from, tc.to) != tc.allowed {
				t.Fatal("incorrect transition")
			}
		})
	}
}
