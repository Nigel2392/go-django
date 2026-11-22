//go:build test

package payments

import (
	"context"
	"database/sql"
	"errors"
	"github.com/Nigel2392/go-django/contrib/shop/shared/signals"
	"github.com/Nigel2392/go-django/contrib/shop/util/paystate"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/shopspring/decimal"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testServiceFailures(t *testing.T, ctx context.Context, psp *servicePSP, account *ProviderAccount, seed func(string) *Payment) {
	refund := func(key, amount string) RefundRequest {
		return RefundRequest{OperationRequest: OperationRequest{RequestID: key}, Amount: Money{Amount: decimal.RequireFromString(amount), Currency: "EUR"}}
	}

	save := func(p *Payment, fields ...any) {
		t.Helper()
		if _, err := queries.GetQuerySet(&Payment{}).Filter("ID", p.ID).Select(fields...).Update(p); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("validation", func(t *testing.T) {
		p := seed(paystate.STATUS_PAID)
		for _, tc := range []struct{ name, key, amount, currency, resource string }{
			{"empty key", "", "1", "EUR", ""}, {"blank key", "  ", "1", "EUR", ""}, {"long key", strings.Repeat("x", 129), "1", "EUR", ""},
			{"zero", "zero", "0", "EUR", ""}, {"negative", "negative", "-1", "EUR", ""}, {"fractional cent", "precision", "0.001", "EUR", ""},
			{"wrong currency", "currency", "1", "USD", ""}, {"invalid currency", "badcurrency", "1", "EURO", ""}, {"excess", "excess", "10.01", "EUR", ""},
			{"wrong resource", "resource", "1", "EUR", "other-payment"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				before := psp.calls
				r := refund(tc.key, tc.amount)
				r.Amount.Currency = tc.currency
				r.ProviderRef = tc.resource
				if _, err := Refund(ctx, p.ID, r); err == nil || psp.calls != before {
					t.Fatal("invalid request submitted", err)
				}

				saved, err := getAttempt(ctx, p.ID)
				if err != nil || saved.OperationKey != "" || len(saved.Operations.Data) != 0 {
					t.Fatal("invalid request changed ledger", err)
				}
			})
		}
	})
	t.Run("invalid payment states", func(t *testing.T) {
		for _, state := range []string{paystate.STATUS_INIT, paystate.STATUS_PENDING, paystate.STATUS_AUTHORIZE, paystate.STATUS_REFUND, paystate.STATUS_FAILURE, paystate.STATUS_CHARGEBACK} {
			t.Run("refund "+state, func(t *testing.T) {
				p := seed(state)
				before := psp.calls
				if _, err := Refund(ctx, p.ID, refund("wrong-state", "1")); err == nil || psp.calls != before {
					t.Fatal("refunded invalid state", err)
				}
			})
		}

		for _, state := range []string{paystate.STATUS_PAID, paystate.STATUS_CANCEL, paystate.STATUS_PENDING} {
			t.Run("authorization required "+state, func(t *testing.T) {
				p := seed(state)
				before := psp.calls
				if _, err := Capture(ctx, p.ID, OperationRequest{RequestID: "invalid"}); err == nil {
					t.Fatal("capture accepted")
				}

				if err := Void(ctx, p.ID, OperationRequest{RequestID: "invalid"}); err == nil || psp.calls != before {
					t.Fatal("void accepted")
				}
			})
		}
	})
	t.Run("unsupported missing and changed provider", func(t *testing.T) {
		basic, err := queries.GetQuerySet(&ProviderAccount{}).Create(&ProviderAccount{Driver: "basic-test", Enabled: true, Currency: "EUR"})
		if err != nil {
			t.Fatal(err)
		}

		p := seed(paystate.STATUS_PAID)
		p.Account = basic
		p.Provider = basic.Driver
		save(p, "Account", "Provider")
		for _, kind := range []string{"refund", "void", "capture", "cancel"} {
			if _, err := invokeServiceOperation(ctx, p.ID, kind, OperationRequest{RequestID: "unsupported"}, refund("", "1").Amount); !errors.Is(err, ErrUnsupportedOperation) {
				t.Fatal(kind, err)
			}
		}

		p.Account = nil
		save(p, "Account")
		if _, err := Refund(ctx, p.ID, refund("no-account", "1")); err == nil {
			t.Fatal("accepted missing account")
		}

		p = seed(paystate.STATUS_PAID)
		p.Provider = "changed"
		save(p, "Provider")
		if _, err := Refund(ctx, p.ID, refund("changed", "1")); err == nil {
			t.Fatal("accepted changed driver")
		}

		if _, err := Refund(ctx, drivers.NewULID(), refund("missing", "1")); err == nil {
			t.Fatal("accepted missing payment")
		}
	})
	t.Run("disabled account refunds and checkout", func(t *testing.T) {
		account.Enabled = false
		if _, err := queries.GetQuerySet(&ProviderAccount{}).Filter("ID", account.ID).Select("Enabled").Update(account); err != nil {
			t.Fatal(err)
		}

		defer func() {
			account.Enabled = true
			queries.GetQuerySet(&ProviderAccount{}).Filter("ID", account.ID).Select("Enabled").Update(account)
		}()
		p := seed(paystate.STATUS_PAID)
		if _, err := Refund(ctx, p.ID, refund("disabled-refund", "1")); err != nil {
			t.Fatal(err)
		}

		p = seed(paystate.STATUS_INIT)
		p.TransactionID = sql.NullString{}
		save(p, "TransactionID")
		before := psp.creates
		if _, err := Start(ctx, p.ID, "https://return", "https://callback"); err == nil || psp.creates != before {
			t.Fatal("disabled account started checkout")
		}
	})
	t.Run("ambiguous actions and reconciliation validation", func(t *testing.T) {
		for _, kind := range []string{"capture", "void", "cancel"} {
			t.Run(kind, func(t *testing.T) {
				state := paystate.STATUS_AUTHORIZE
				if kind == "cancel" {
					state = paystate.STATUS_PENDING
				}

				p := seed(state)
				psp.err = errors.New("connection lost")
				_, err := invokeServiceOperation(ctx, p.ID, kind, OperationRequest{RequestID: "uncertain"}, Money{})
				psp.err = nil
				if err == nil {
					t.Fatal("ignored failure")
				}

				before := psp.calls
				if _, err := invokeServiceOperation(ctx, p.ID, kind, OperationRequest{RequestID: "uncertain"}, Money{}); !errors.Is(err, ErrOperationPending) || psp.calls != before {
					t.Fatal("uncertain operation retried", err)
				}

				for _, tc := range []struct {
					key    string
					result OperationResult
				}{{"unknown", OperationResult{State: "succeeded", ProviderRef: "op"}}, {"uncertain", OperationResult{State: "pending"}}, {"uncertain", OperationResult{State: "succeeded"}}} {
					if _, err := ResolveOperation(ctx, p.ID, tc.key, tc.result); err == nil {
						t.Fatal("invalid reconciliation accepted", tc)
					}
				}

				if _, err := ResolveOperation(ctx, p.ID, "uncertain", OperationResult{State: "succeeded", ProviderRef: "op"}); err != nil {
					t.Fatal(err)
				}
			})
		}
	})
	t.Run("completion listener failure preserves reservation", func(t *testing.T) {
		p := seed(paystate.STATUS_PAID)
		receiver, err := Signals.Refund.Listen(ctx, func(context.Context, signals.Signal[PaymentRefundedSignal], PaymentRefundedSignal) error {
			return errors.New("downstream failure")
		})
		if err != nil {
			t.Fatal(err)
		}

		before := psp.calls
		_, err = Refund(ctx, p.ID, refund("listener-error", "2"))
		Signals.Refund.Disconnect(ctx, receiver)
		if err == nil || psp.calls != before+1 {
			t.Fatal("expected signal failure", err)
		}

		if _, err := Refund(ctx, p.ID, refund("listener-error", "2")); !errors.Is(err, ErrOperationPending) || psp.calls != before+1 {
			t.Fatal("retried remote refund", err)
		}

		if _, err := ResolveOperation(ctx, p.ID, "listener-error", psp.result); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("transaction contexts", func(t *testing.T) {
		p := seed(paystate.STATUS_PAID)
		before := psp.calls
		if _, err := Refund(queries.CommitContext(ctx, false), p.ID, refund("rollback", "1")); err == nil {
			t.Fatal("rollback-only operation")
		}

		txctx, tx, err := queries.StartTransaction(ctx)
		if err != nil {
			t.Fatal(err)
		}

		defer tx.Rollback(txctx)
		if _, err := Refund(txctx, p.ID, refund("nested", "1")); err == nil || psp.calls != before {
			t.Fatal("nested transaction operation", err)
		}
	})
	t.Run("checkout refresh callbacks and signals", func(t *testing.T) {
		starts, updates := 0, 0
		sr, err := Signals.Start.Listen(ctx, func(_ context.Context, _ signals.Signal[PaymentStartedSignal], s PaymentStartedSignal) error {
			starts++
			if s.Payment == nil {
				t.Error("missing signal data")
			}

			return nil
		})
		if err != nil {
			t.Fatal(err)
		}

		defer Signals.Start.Disconnect(ctx, sr)
		ur, err := Signals.Update.Listen(ctx, func(context.Context, signals.Signal[PaymentUpdatedSignal], PaymentUpdatedSignal) error {
			updates++
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}

		defer Signals.Update.Disconnect(ctx, ur)
		p := seed(paystate.STATUS_INIT)
		p.TransactionID = sql.NullString{}
		save(p, "TransactionID")
		psp.status = PaymentStatus{ProviderRef: "checkout", State: paystate.STATUS_PENDING, Amount: Money{Amount: p.Amount, Currency: p.Currency}}
		before := psp.creates
		for i := 0; i < 2; i++ {
			if _, err := Start(ctx, p.ID, "https://return", "https://callback"); err != nil {
				t.Fatal(err)
			}
		}

		if starts != 1 || psp.creates != before+1 {
			t.Fatal("duplicate start")
		}

		psp.status.State = paystate.STATUS_PAID
		before = updates
		for i := 0; i < 2; i++ {
			if err := reconcileWebhook(ctx, account, psp.status); err != nil {
				t.Fatal(err)
			}
		}

		if updates != before+1 {
			t.Fatal("duplicate callback signal")
		}

		for _, bad := range []PaymentStatus{{ProviderRef: "", State: paystate.STATUS_PAID}, {ProviderRef: "checkout", State: paystate.STATUS_PAID, Amount: Money{Amount: decimal.NewFromInt(1), Currency: "EUR"}}} {
			if err := reconcileWebhook(ctx, account, bad); err == nil {
				t.Fatal("invalid callback accepted")
			}
		}

		psp.err = errors.New("offline")
		if _, err := Refresh(ctx, p.ID); err == nil {
			t.Fatal("ignored refresh failure")
		}

		psp.err = nil
		p = seed(paystate.STATUS_INIT)
		p.TransactionID = sql.NullString{}
		save(p, "TransactionID")
		before = psp.gets
		if _, err := Refresh(ctx, p.ID); err != nil || psp.gets != before {
			t.Fatal("refresh requested without transaction")
		}
	})
	t.Run("create attempt and quote failure cases", func(t *testing.T) {
		order, err := queries.GetQuerySet(&Order{}).Create(&Order{ID: drivers.NewULID()})
		if err != nil {
			t.Fatal(err)
		}

		for _, tc := range []struct {
			name    string
			order   *Order
			account drivers.ULID
			amount  Money
		}{
			{"missing account", order, drivers.NewULID(), Money{decimal.NewFromInt(10), "EUR"}},
			{"zero", order, account.ID, Money{decimal.Zero, "EUR"}},
			{"negative", order, account.ID, Money{decimal.NewFromInt(-1), "EUR"}},
			{"currency mismatch", order, account.ID, Money{decimal.NewFromInt(10), "USD"}},
			{"unknown currency", order, account.ID, Money{decimal.NewFromInt(10), "garbage"}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				if _, err := CreateAttempt(ctx, tc.order, tc.account, tc.amount); err == nil {
					t.Fatal("invalid attempt created")
				}
			})
		}

		if _, err := CreateAttempt(ctx, nil, account.ID, Money{decimal.NewFromInt(10), "EUR"}); err == nil {
			t.Fatal("nil order accepted")
		}

		account.FixedFee = decimal.RequireFromString("0.5")
		account.PercentageFee = decimal.NewFromInt(2)
		if _, err := queries.GetQuerySet(&ProviderAccount{}).Filter("ID", account.ID).Select("FixedFee", "PercentageFee").Update(account); err != nil {
			t.Fatal(err)
		}

		p, err := CreateAttempt(ctx, order, account.ID, Money{decimal.NewFromInt(10), "EUR"})
		if err != nil || !p.Amount.Equal(decimal.RequireFromString("10.7")) || !p.Fee.Equal(decimal.RequireFromString("0.7")) || p.Provider != account.Driver {
			t.Fatal("incorrect attempt snapshot", err)
		}

		account.FixedFee = decimal.Zero
		account.PercentageFee = decimal.Zero
		queries.GetQuerySet(&ProviderAccount{}).Filter("ID", account.ID).Select("FixedFee", "PercentageFee").Update(account)
	})
	t.Run("start failures preserve an unopened payment", func(t *testing.T) {
		p := seed(paystate.STATUS_INIT)
		p.TransactionID = sql.NullString{}
		save(p, "TransactionID")
		for _, tc := range []struct {
			name   string
			status PaymentStatus
			err    error
		}{
			{"PSP outage", PaymentStatus{}, errors.New("offline")},
			{"no reference", PaymentStatus{State: paystate.STATUS_PENDING, Amount: Money{p.Amount, p.Currency}}, nil},
			{"unknown status", PaymentStatus{ProviderRef: "invalid", State: "UNKNOWN", Amount: Money{p.Amount, p.Currency}}, nil},
			{"wrong amount", PaymentStatus{ProviderRef: "invalid", State: paystate.STATUS_PENDING, Amount: Money{decimal.NewFromInt(1), "EUR"}}, nil},
			{"wrong currency", PaymentStatus{ProviderRef: "invalid", State: paystate.STATUS_PENDING, Amount: Money{p.Amount, "USD"}}, nil},
		} {
			t.Run(tc.name, func(t *testing.T) {
				psp.status = tc.status
				psp.err = tc.err
				if _, err := Start(ctx, p.ID, "https://return", "https://callback"); err == nil {
					t.Fatal("invalid checkout accepted")
				}

				saved, err := getAttempt(ctx, p.ID)
				if err != nil || saved.TransactionID.Valid || saved.State != paystate.STATUS_INIT {
					t.Fatal("failure changed payment", err)
				}
			})
		}

		psp.err = nil
	})
	t.Run("webhook HTTP errors and successful replay", func(t *testing.T) {
		p := seed(paystate.STATUS_PENDING)
		psp.status = PaymentStatus{ProviderRef: p.TransactionID.String, State: paystate.STATUS_PAID, Amount: Money{p.Amount, p.Currency}}
		for _, tc := range []struct {
			name, account, body string
			err                 error
			code                int
		}{
			{"invalid account", "bad", "", nil, 404},
			{"unknown account", drivers.NewULID().String(), "", nil, 404},
			{"oversized body", account.ID.String(), strings.Repeat("x", (1<<20)+1), nil, 400},
			{"invalid signature", account.ID.String(), "", ErrInvalidWebhook, 400},
			{"provider failure", account.ID.String(), "", errors.New("offline"), 503},
			{"success", account.ID.String(), "", nil, 200},
			{"replay", account.ID.String(), "", nil, 200},
		} {
			t.Run(tc.name, func(t *testing.T) {
				psp.err = tc.err
				rec := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodPost, "/callback", strings.NewReader(tc.body))
				ServeWebhook(rec, r, tc.account)
				if rec.Code != tc.code {
					t.Fatal(rec.Code, rec.Body.String())
				}
			})
		}

		psp.err = nil
	})
	t.Run("refresh persists resource references without duplicate signals", func(t *testing.T) {
		p := seed(paystate.STATUS_AUTHORIZE)
		psp.status = PaymentStatus{ProviderRef: p.TransactionID.String, State: p.State, AuthorizationRef: "auth", CaptureRef: "capture", Amount: Money{p.Amount, p.Currency}}
		for i := 0; i < 2; i++ {
			got, err := Refresh(ctx, p.ID)
			if err != nil || got.AuthorizationRef != "auth" || got.CaptureRef != "capture" {
				t.Fatal("resource references lost", err)
			}
		}
	})
	t.Run("provider failure to resolve and missing transaction", func(t *testing.T) {
		p := seed(paystate.STATUS_PAID)
		p.TransactionID = sql.NullString{}
		save(p, "TransactionID")
		if _, err := Refund(ctx, p.ID, refund("no-transaction", "1")); err == nil {
			t.Fatal("refund without transaction")
		}

		a, err := queries.GetQuerySet(&ProviderAccount{}).Create(&ProviderAccount{Driver: "missing-adapter", Currency: "EUR"})
		if err != nil {
			t.Fatal(err)
		}

		p = seed(paystate.STATUS_PAID)
		p.Account = a
		p.Provider = a.Driver
		save(p, "Account", "Provider")
		if _, err := Refund(ctx, p.ID, refund("no-provider", "1")); err == nil {
			t.Fatal("missing provider accepted")
		}
	})
	t.Run("start and update listener failures roll back local state", func(t *testing.T) {
		p := seed(paystate.STATUS_INIT)
		p.TransactionID = sql.NullString{}
		save(p, "TransactionID")
		psp.status = PaymentStatus{ProviderRef: "signal-checkout", State: paystate.STATUS_PENDING, Amount: Money{p.Amount, p.Currency}}
		sr, err := Signals.Start.Listen(ctx, func(context.Context, signals.Signal[PaymentStartedSignal], PaymentStartedSignal) error {
			return errors.New("start handler failed")
		})
		if err != nil {
			t.Fatal(err)
		}

		_, err = Start(ctx, p.ID, "return", "callback")
		Signals.Start.Disconnect(ctx, sr)
		if err == nil {
			t.Fatal("start listener error ignored")
		}

		saved, err := getAttempt(ctx, p.ID)
		if err != nil || saved.TransactionID.Valid {
			t.Fatal("failed start signal committed", err)
		}

		if _, err := Start(ctx, p.ID, "return", "callback"); err != nil {
			t.Fatal(err)
		}

		ur, err := Signals.Update.Listen(ctx, func(context.Context, signals.Signal[PaymentUpdatedSignal], PaymentUpdatedSignal) error {
			return errors.New("update handler failed")
		})
		if err != nil {
			t.Fatal(err)
		}

		psp.status.State = paystate.STATUS_PAID
		_, err = Refresh(ctx, p.ID)
		Signals.Update.Disconnect(ctx, ur)
		if err == nil {
			t.Fatal("update listener error ignored")
		}

		saved, err = getAttempt(ctx, p.ID)
		if err != nil || saved.State != paystate.STATUS_PENDING {
			t.Fatal("failed update signal committed", err)
		}

		if _, err := Refresh(ctx, p.ID); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("pending callback cannot bypass operation reservation", func(t *testing.T) {
		p := seed(paystate.STATUS_PAID)
		psp.result.State = "pending"
		if _, err := Refund(ctx, p.ID, refund("pending-callback", "1")); err != nil {
			t.Fatal(err)
		}

		psp.result.State = "succeeded"
		status := PaymentStatus{ProviderRef: p.TransactionID.String, State: paystate.STATUS_REFUND, Amount: Money{p.Amount, p.Currency}}
		if err := reconcileWebhook(ctx, account, status); !errors.Is(err, ErrOperationPending) {
			t.Fatal("callback bypassed reservation", err)
		}

		if _, err := ResolveOperation(ctx, p.ID, "pending-callback", OperationResult{State: "failed"}); err != nil {
			t.Fatal(err)
		}

		if _, err := ResolveOperation(ctx, p.ID, "pending-callback", OperationResult{State: "succeeded", ProviderRef: "contradiction"}); err == nil {
			t.Fatal("terminal outcome overwritten")
		}
	})
	t.Run("adapter-specific resource binding uses capabilities", func(t *testing.T) {
		RegisterProvider("reference-test", func(context.Context, *ProviderAccount) (Provider, error) { return referencePSP{psp}, nil })
		a, err := queries.GetQuerySet(&ProviderAccount{}).Create(&ProviderAccount{Driver: "reference-test", Currency: "EUR"})
		if err != nil {
			t.Fatal(err)
		}

		p := seed(paystate.STATUS_PAID)
		p.Account = a
		p.Provider = a.Driver
		save(p, "Account", "Provider")
		if _, err := Refund(ctx, p.ID, refund("missing-capture", "1")); err == nil {
			t.Fatal("order ID used as capture ID")
		}

		p.CaptureRef = "capture-resource"
		save(p, "CaptureRef")
		r := refund("bound-refund", "1")
		r.ProviderRef = p.TransactionID.String
		if _, err := Refund(ctx, p.ID, r); err == nil {
			t.Fatal("wrong adapter resource accepted")
		}

		r.ProviderRef = p.CaptureRef
		if _, err := Refund(ctx, p.ID, r); err != nil {
			t.Fatal(err)
		}

		p = seed(paystate.STATUS_AUTHORIZE)
		p.Account = a
		p.Provider = a.Driver
		p.AuthorizationRef = "auth-resource"
		save(p, "Account", "Provider", "AuthorizationRef")
		if _, err := Capture(ctx, p.ID, OperationRequest{RequestID: "bound-capture", ProviderRef: p.AuthorizationRef}); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("legacy nullable operation metadata", func(t *testing.T) {
		p := seed(paystate.STATUS_PAID)
		if _, err := dbExec(ctx, "UPDATE shop_payment SET operation_key=NULL,operation_version=NULL,operations=NULL WHERE id=?", p.ID); err != nil {
			t.Fatal(err)
		}

		if _, err := Refund(ctx, p.ID, refund("legacy", "1")); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("simultaneous full refunds mutate provider once", func(t *testing.T) {
		p := seed(paystate.STATUS_PAID)
		before := psp.calls
		gate := make(chan struct{})
		done := make(chan error, 2)
		for _, key := range []string{"first-concurrent", "second-concurrent"} {
			go func(key string) {
				<-gate

				_, err := Refund(ctx, p.ID, refund(key, "10"))
				done <- err
			}(key)
		}

		close(gate)
		successes := 0
		for i := 0; i < 2; i++ {
			if <-done == nil {
				successes++
			}
		}

		if successes > 1 || psp.calls != before+1 {
			t.Fatal("concurrent refunds exceeded balance", successes, psp.calls-before)
		}

		saved, err := getAttempt(ctx, p.ID)
		if err != nil {
			t.Fatal(err)
		}

		// Local completion can lose a database lock race after the remote
		// mutation. Its durable reservation must allow reconciliation.
		if saved.OperationKey != "" {
			if _, err := ResolveOperation(ctx, p.ID, saved.OperationKey, psp.result); err != nil {
				t.Fatal("reserved refund could not be reconciled", err)
			}

			saved, err = getAttempt(ctx, p.ID)
		}

		if err != nil || saved.State != paystate.STATUS_REFUND {
			t.Fatal("incorrect final refund state", err)
		}
	})
	t.Run("provider capability discovery", func(t *testing.T) {
		provider, err := account.Provider(ctx)
		if err != nil {
			t.Fatal(err)
		}

		if _, ok := provider.(Refunder); !ok {
			t.Fatal("refund capability lost")
		}

		if !ProviderRegistered(account.Driver) || ProviderRegistered("unknown-provider") {
			t.Fatal("provider registration lookup")
		}

		if err := (OperationRequest{}).Validate(); err == nil {
			t.Fatal("invalid resource accepted")
		}

		if err := (RefundRequest{Amount: Money{decimal.NewFromInt(1), "EUR"}}).Validate(); err == nil {
			t.Fatal("refund missing operation identity")
		}
	})
}

func dbExec(ctx context.Context, statement string, args ...any) (sql.Result, error) {
	return queries.GetQuerySet(&Payment{}).WithContext(ctx).DB().ExecContext(ctx, statement, args...)
}

// Tests dispatch by operation to exercise the public service entry points.
func invokeServiceOperation(ctx context.Context, id drivers.ULID, kind string, r OperationRequest, amount Money) (OperationResult, error) {
	switch kind {
	case "refund":
		return Refund(ctx, id, RefundRequest{OperationRequest: r, Amount: amount})
	case "capture":
		return Capture(ctx, id, r)
	case "void":
		return OperationResult{}, Void(ctx, id, r)
	case "cancel":
		return OperationResult{}, Cancel(ctx, id, r)
	default:
		panic("unknown test operation")
	}
}

type referencePSP struct{ *servicePSP }

func (p referencePSP) OperationReferences(status PaymentStatus) OperationReferences {
	return OperationReferences{Refund: status.CaptureRef, Void: status.AuthorizationRef, Capture: status.AuthorizationRef}
}
