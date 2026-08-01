package paystate

import (
	"fmt"
	"testing"
)

var testStatusTranslationsTable = [][2]PayState{
	{STATUS_INIT, "Initialised"},
	{STATUS_PENDING, "Pending"},
	{STATUS_CANCEL, "Cancelled"},
	{STATUS_EXPIRED, "Expired"},
	{STATUS_DENIED, "Denied"},
	{STATUS_FAILURE, "Failure"},
	{STATUS_PAID_CHECKAMOUNT, "Check Amount"},
	{STATUS_PARTIAL_PAYMENT, "Partial Payment"},
	{STATUS_VERIFY, "Manual Verify"},
	{STATUS_AUTHORIZE, "Authorize"},
	{STATUS_PARTLY_CAPTURED, "Partly Captured"},
	{STATUS_PAID, "Paid"},
	{STATUS_CHARGEBACK, "Chargeback"},
	{STATUS_REFUNDING, "Refunding"},
	{STATUS_REFUND, "Refunded"},
	{STATUS_PARTIAL_REFUND, "Partial Refund"},

	{STATUS_INIT + ":" + STATUS_PENDING, "Initialised | Pending"},
	{STATUS_CANCEL + ":" + STATUS_EXPIRED, "Cancelled | Expired"},
	{STATUS_DENIED + ":" + STATUS_FAILURE, "Denied | Failure"},
	{STATUS_PAID_CHECKAMOUNT + ":" + STATUS_PARTIAL_PAYMENT, "Check Amount | Partial Payment"},
	{STATUS_VERIFY + ":" + STATUS_AUTHORIZE, "Manual Verify | Authorize"},
	{STATUS_PARTLY_CAPTURED + ":" + STATUS_PAID, "Partly Captured | Paid"},
	{STATUS_CHARGEBACK + ":" + STATUS_REFUNDING, "Chargeback | Refunding"},
	{STATUS_REFUND + ":" + STATUS_PARTIAL_REFUND, "Refunded | Partial Refund"},
}

func TestStatusText(t *testing.T) {
	for _, obj := range testStatusTranslationsTable {
		t.Run(fmt.Sprintf("TestTranslationStatus(%q)", obj[0]), func(t *testing.T) {
			var (
				status = obj[0]
				trans  = obj[1]
			)

			if chkStatus := PAYSTATE.StringOf(t.Context(), status); chkStatus != string(trans) {
				t.Fatalf("%q did not match proper translation: %q", status, chkStatus)
			}
		})
	}
}
