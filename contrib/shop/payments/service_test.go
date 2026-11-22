//go:build test

package payments

import (
	"context"
	"database/sql"
	"errors"
	"github.com/Nigel2392/go-django/contrib/shop/shared/signals"
	"github.com/Nigel2392/go-django/contrib/shop/util/paystate"
	"github.com/Nigel2392/go-django/djester/quest"
	"github.com/Nigel2392/go-django/djester/testdb"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/shopspring/decimal"
	"testing"
)

type Order struct {
	ID drivers.ULID `attrs:"primary"`
}

func (o *Order) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.AutoDefinitions(ctx, o)
}

type servicePSP struct {
	creates, gets int
	calls         int
	result        OperationResult
	err           error
	during        func()
	status        PaymentStatus
}

func (p *servicePSP) CreatePayment(context.Context, CreatePaymentRequest) (Checkout, error) {
	p.creates++
	return Checkout{PaymentStatus: p.status, URL: "https://psp.invalid"}, p.err
}

func (p *servicePSP) GetPayment(context.Context, string) (PaymentStatus, error) {
	p.gets++
	return p.status, p.err
}

func (p *servicePSP) HandleWebhook(context.Context, WebhookRequest) (WebhookResult, error) {
	return WebhookResult{Payment: p.status}, p.err
}

func (p *servicePSP) operation() (OperationResult, error) {
	p.calls++
	if p.during != nil {
		p.during()
	}

	return p.result, p.err
}

func (p *servicePSP) RefundPayment(context.Context, RefundRequest) (OperationResult, error) {
	return p.operation()
}

func (p *servicePSP) CapturePayment(context.Context, OperationRequest) (OperationResult, error) {
	return p.operation()
}

func (p *servicePSP) VoidPayment(context.Context, OperationRequest) error {
	_, err := p.operation()
	return err
}

func (p *servicePSP) CancelPayment(context.Context, OperationRequest) error {
	_, err := p.operation()
	return err
}

type basicPSP struct{ Provider }

func TestServiceOperations(t *testing.T) {
	ctx := context.Background()
	_, db := testdb.Open()
	defer db.Close()
	app := django.App(django.Configure(map[string]any{django.APPVAR_DATABASE: db}), django.Flag(django.FlagSkipDepsCheck, django.FlagSkipChecks, django.FlagSkipCmds), django.Apps(func() django.AppConfig {
		cfg := apps.NewDBAppConfig("shop")
		cfg.ModelObjects = []attrs.Definer{&Order{}, &ProviderAccount{}, &Payment{}}
		return cfg
	}))
	if err := app.Initialize(); err != nil {
		t.Fatal(err)
	}

	quest.Table(t, &Order{}, &ProviderAccount{}, &Payment{}).Create()
	psp := &servicePSP{result: OperationResult{ProviderRef: "receipt", Status: "COMPLETED", State: "succeeded"}}
	RegisterProvider("service-test", func(context.Context, *ProviderAccount) (Provider, error) { return psp, nil })
	RegisterProvider("basic-test", func(context.Context, *ProviderAccount) (Provider, error) { return basicPSP{psp}, nil })
	account, err := queries.GetQuerySet(&ProviderAccount{}).Create(&ProviderAccount{Driver: "service-test", Enabled: true, Currency: "EUR", Label: "test"})
	if err != nil {
		t.Fatal(err)
	}

	counts := map[string]int{}
	listen := func(name string, s signals.Signal[PaymentOperationSignal]) {
		recv, err := s.Listen(ctx, func(_ context.Context, _ signals.Signal[PaymentOperationSignal], v PaymentOperationSignal) error {
			if v.Payment == nil || v.RequestID == "" {
				t.Error("missing operation signal data")
			}

			counts[name]++
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}

		t.Cleanup(func() { s.Disconnect(ctx, recv) })
	}

	listen("requested", Signals.OperationRequested)
	listen("pending", Signals.OperationPending)
	listen("failed", Signals.OperationFailed)
	listen("capture", Signals.Capture)
	listen("cancel", Signals.Cancel)
	refunds := 0
	recv, err := Signals.Refund.Listen(ctx, func(_ context.Context, _ signals.Signal[PaymentRefundedSignal], s PaymentRefundedSignal) error {
		refunds++
		if s.Payment == nil || !s.Amount.Amount.IsPositive() {
			t.Error("refund signal payload")
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	defer Signals.Refund.Disconnect(ctx, recv)
	voids := 0
	vr, err := Signals.Void.Listen(ctx, func(_ context.Context, _ signals.Signal[PaymentVoidedSignal], s PaymentVoidedSignal) error {
		voids++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	defer Signals.Void.Disconnect(ctx, vr)
	updates := 0
	ur, err := Signals.Update.Listen(ctx, func(_ context.Context, _ signals.Signal[PaymentUpdatedSignal], s PaymentUpdatedSignal) error {
		updates++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	defer Signals.Update.Disconnect(ctx, ur)
	seed := func(state string) *Payment {
		p, err := queries.GetQuerySet(&Payment{}).Create(&Payment{Account: account, Provider: account.Driver, TransactionID: sql.NullString{String: drivers.NewULID().String(), Valid: true}, Amount: decimal.NewFromInt(10), Currency: "EUR", State: state})
		if err != nil {
			t.Fatal(err)
		}

		return p
	}

	refund := func(key, amount string) RefundRequest {
		return RefundRequest{OperationRequest: OperationRequest{RequestID: key}, Amount: Money{Amount: decimal.RequireFromString(amount), Currency: "EUR"}}
	}

	t.Run("partial full refund and replay", func(t *testing.T) {
		p := seed(paystate.STATUS_PAID)
		before := psp.calls
		for i := 0; i < 2; i++ {
			if _, err := Refund(ctx, p.ID, refund("partial", "3")); err != nil {
				t.Fatal(err)
			}
		}

		if psp.calls != before+1 || refunds != 1 {
			t.Fatal("duplicate refund or signal")
		}

		if _, err := Refund(ctx, p.ID, refund("partial", "4")); err == nil {
			t.Fatal("request reused with changed amount")
		}

		if _, err := Refund(ctx, p.ID, refund("excess", "8")); err == nil {
			t.Fatal("over-refund")
		}

		if _, err := Refund(ctx, p.ID, refund("remaining", "7")); err != nil {
			t.Fatal(err)
		}

		saved, err := getAttempt(ctx, p.ID)
		if err != nil || saved.State != paystate.STATUS_REFUND {
			t.Fatal("full refund state", err)
		}
	})
	t.Run("capture void cancel and invalid states", func(t *testing.T) {
		p := seed(paystate.STATUS_AUTHORIZE)
		if _, err := Capture(ctx, p.ID, OperationRequest{RequestID: "capture"}); err != nil {
			t.Fatal(err)
		}

		if _, err := Capture(ctx, p.ID, OperationRequest{RequestID: "capture"}); err != nil {
			t.Fatal("replay", err)
		}

		if err := Void(ctx, p.ID, OperationRequest{RequestID: "invalid"}); err == nil {
			t.Fatal("voided captured payment")
		}

		p = seed(paystate.STATUS_AUTHORIZE)
		if err := Void(ctx, p.ID, OperationRequest{RequestID: "void"}); err != nil {
			t.Fatal(err)
		}

		p = seed(paystate.STATUS_PENDING)
		if err := Cancel(ctx, p.ID, OperationRequest{RequestID: "cancel"}); err != nil {
			t.Fatal(err)
		}

		if counts["capture"] != 1 || counts["cancel"] != 1 || voids != 1 {
			t.Fatal("missing success signals", counts, voids)
		}
	})
	t.Run("pending reserves balance and reconciliation is idempotent", func(t *testing.T) {
		p := seed(paystate.STATUS_PAID)
		psp.result.State = "pending"
		defer func() { psp.result.State = "succeeded" }()
		if _, err := Refund(ctx, p.ID, refund("async", "6")); err != nil {
			t.Fatal(err)
		}

		before := psp.calls
		if _, err := Refund(ctx, p.ID, refund("async", "6")); !errors.Is(err, ErrOperationPending) {
			t.Fatal(err)
		}

		if _, err := Refund(ctx, p.ID, refund("other", "5")); !errors.Is(err, ErrOperationPending) {
			t.Fatal(err)
		}

		if psp.calls != before {
			t.Fatal("pending operation resubmitted")
		}

		for i := 0; i < 2; i++ {
			if _, err := ResolveOperation(ctx, p.ID, "async", OperationResult{State: "succeeded", ProviderRef: "receipt"}); err != nil {
				t.Fatal(err)
			}
		}

		if _, err := Refund(ctx, p.ID, refund("too-much", "5")); err == nil {
			t.Fatal("refund balance lost")
		}
	})
	t.Run("ambiguous failure prevents duplicate requests", func(t *testing.T) {
		p := seed(paystate.STATUS_PAID)
		psp.err = errors.New("response lost")
		if _, err := Refund(ctx, p.ID, refund("lost", "2")); err == nil {
			t.Fatal("ignored PSP failure")
		}

		psp.err = nil
		before := psp.calls
		if _, err := Refund(ctx, p.ID, refund("lost", "2")); !errors.Is(err, ErrOperationPending) || psp.calls != before {
			t.Fatal("duplicate mutation", err)
		}

		if _, err := ResolveOperation(ctx, p.ID, "lost", OperationResult{State: "failed"}); err != nil {
			t.Fatal(err)
		}

		if _, err := Refund(ctx, p.ID, refund("new", "10")); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("concurrent request cannot spend reserved balance", func(t *testing.T) {
		p := seed(paystate.STATUS_PAID)
		psp.during = func() {
			if _, err := Refund(ctx, p.ID, refund("competing", "10")); !errors.Is(err, ErrOperationPending) {
				t.Error("reservation bypassed", err)
			}
		}

		defer func() { psp.during = nil }()
		if _, err := Refund(ctx, p.ID, refund("winner", "10")); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("signals veto before provider call", func(t *testing.T) {
		receiver, err := Signals.OperationRequested.Listen(ctx, func(context.Context, signals.Signal[PaymentOperationSignal], PaymentOperationSignal) error {
			return errors.New("veto")
		})
		if err != nil {
			t.Fatal(err)
		}

		defer Signals.OperationRequested.Disconnect(ctx, receiver)
		p := seed(paystate.STATUS_PAID)
		before := psp.calls
		if _, err := Refund(ctx, p.ID, refund("veto", "1")); err == nil || psp.calls != before {
			t.Fatal("signal veto ignored")
		}

		saved, err := getAttempt(ctx, p.ID)
		if err != nil || saved.OperationKey != "" {
			t.Fatal("reservation rollback failed", err)
		}
	})
	testServiceFailures(t, ctx, psp, account, seed)
	if updates == 0 || counts["pending"] == 0 || counts["failed"] == 0 {
		t.Fatal("lifecycle signals missing", counts)
	}
}
