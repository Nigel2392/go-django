//go:build test

package payments

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/contrib/shop/shared/price"
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
	"github.com/stretchr/testify/require"
	"golang.org/x/text/currency"
)

type Order struct {
	ID drivers.ULID `attrs:"primary"`
}

func (o *Order) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.AutoDefinitions(ctx, o).(*attrs.ObjectDefinitions).
		WithTableName("payment_service_test_order")
}

type servicePSP struct {
	creates, gets  int
	refunds        int
	cancels        int
	refundResult   RefundResult
	err            error
	status         PaymentStatus
	createStatusFn func(CreatePaymentRequest) PaymentStatus
	acknowledge    func(http.ResponseWriter)
}

func (p *servicePSP) CreatePayment(_ context.Context, req CreatePaymentRequest) (Checkout, error) {
	p.creates++
	status := p.status
	if p.createStatusFn != nil {
		status = p.createStatusFn(req)
	}
	return Checkout{PaymentStatus: status, URL: "https://psp.invalid/pay/" + req.AttemptID.String()}, p.err
}

func (p *servicePSP) GetPayment(context.Context, string) (PaymentStatus, error) {
	p.gets++
	return p.status, p.err
}

func (p *servicePSP) HandleWebhook(context.Context, WebhookRequest) (WebhookResult, error) {
	return WebhookResult{Payment: p.status, Acknowledge: p.acknowledge}, p.err
}

func (p *servicePSP) RefundPayment(_ context.Context, _ RefundRequest) (RefundResult, error) {
	p.refunds++
	return p.refundResult, p.err
}

func (p *servicePSP) CancelPayment(_ context.Context, _ CancelRequest) error {
	p.cancels++
	return p.err
}

type basicPSP struct{ Provider }

func TestServiceOperations(t *testing.T) {
	ctx := context.Background()
	_, db := testdb.Open()
	defer db.Close()
	app := django.App(
		django.Configure(map[string]any{
			django.APPVAR_DATABASE: db,
		}),
		django.Flag(django.FlagSkipDepsCheck, django.FlagSkipChecks, django.FlagSkipCmds),
		django.Apps(
			func() django.AppConfig {
				cfg := apps.NewDBAppConfig("shop")
				cfg.ModelObjects = []attrs.Definer{&Order{}}
				return cfg
			},
			func() django.AppConfig {
				cfg := apps.NewDBAppConfig("payment_service_test")
				cfg.ModelObjects = []attrs.Definer{&ProviderAccount{}, &Payment{}}
				return cfg
			},
		),
	)
	require.NoError(t, app.Initialize())

	tables := quest.Table(t, &Order{}, &ProviderAccount{}, &Payment{})
	tables.Create()
	defer tables.Drop()

	psp := &servicePSP{refundResult: RefundResult{ProviderRef: "refund-receipt"}}
	basePSP := &PSP{
		Name:              "service-test",
		CurrencyUnit:      currency.EUR,
		FixedFeeAmount:    decimal.RequireFromString("0.50"),
		PercentageFeeRate: decimal.NewFromInt(2),
		TextLabel:         func(context.Context) string { return "Service Test" },
		Factory:           func(context.Context, *ProviderAccount) (Provider, error) { return psp, nil },
	}
	RegisterProvider(basePSP)
	RegisterProvider(&PSP{
		Name:         "basic-test",
		CurrencyUnit: currency.EUR,
		Factory:      func(context.Context, *ProviderAccount) (Provider, error) { return basicPSP{psp}, nil },
	})

	account, err := queries.GetQuerySet(&ProviderAccount{}).Create(&ProviderAccount{
		InternalName: "service-test",
		Enabled:      true,
	})
	require.NoError(t, err)

	starts, updates, refundSignals, cancelSignals := 0, 0, 0, 0
	sr, err := Signals.Start.Listen(ctx, func(_ context.Context, _ signals.Signal[PaymentStartedSignal], s PaymentStartedSignal) error {
		starts++
		require.NotNil(t, s.Payment)
		return nil
	})
	require.NoError(t, err)
	defer Signals.Start.Disconnect(ctx, sr)

	ur, err := Signals.Update.Listen(ctx, func(_ context.Context, _ signals.Signal[PaymentUpdatedSignal], s PaymentUpdatedSignal) error {
		updates++
		require.NotNil(t, s.Payment)
		return nil
	})
	require.NoError(t, err)
	defer Signals.Update.Disconnect(ctx, ur)

	rr, err := Signals.Refund.Listen(ctx, func(_ context.Context, _ signals.Signal[PaymentRefundedSignal], s PaymentRefundedSignal) error {
		refundSignals++
		require.NotNil(t, s.Payment)
		require.True(t, s.Amount.Amount.IsPositive())
		require.Equal(t, currency.EUR, s.Amount.Currency)
		return nil
	})
	require.NoError(t, err)
	defer Signals.Refund.Disconnect(ctx, rr)

	cr, err := Signals.Cancel.Listen(ctx, func(_ context.Context, _ signals.Signal[PaymentCancelledSignal], s PaymentCancelledSignal) error {
		cancelSignals++
		require.NotNil(t, s.Payment)
		require.NotEmpty(t, s.RequestID)
		return nil
	})
	require.NoError(t, err)
	defer Signals.Cancel.Disconnect(ctx, cr)

	seedOrder := func() *Order {
		o, err := queries.GetQuerySet(&Order{}).Create(&Order{ID: drivers.NewULID()})
		require.NoError(t, err)
		return o
	}

	seedPayment := func(state string) *Payment {
		p, err := queries.GetQuerySet(&Payment{}).Create(&Payment{
			Order:         seedOrder(),
			Account:       account,
			Provider:      account.InternalName,
			TransactionID: sql.NullString{String: drivers.NewULID().String(), Valid: true},
			Amount:        decimal.NewFromInt(10),
			Fee:           decimal.Zero,
			Currency:      "EUR",
			State:         state,
		})
		require.NoError(t, err)
		return p
	}

	t.Run("OpenCheckout performs zero DB writes on failure and RecordPayment persists opened checkout", func(t *testing.T) {
		countBefore, err := queries.GetQuerySet(&Payment{}).Count()
		require.NoError(t, err)

		attemptID := drivers.NewULID()
		subtotal := price.Money{Amount: decimal.NewFromInt(10), Currency: currency.EUR}

		// Input & quote validation failures
		_, _, err = OpenCheckout(ctx, account, drivers.ULID{}, subtotal, "https://ret", "https://wh")
		require.Error(t, err)

		_, _, err = OpenCheckout(ctx, nil, attemptID, subtotal, "https://ret", "https://wh")
		require.Error(t, err)

		_, _, err = OpenCheckout(ctx, &ProviderAccount{ID: drivers.NewULID(), InternalName: "service-test", Enabled: false}, attemptID, subtotal, "https://ret", "https://wh")
		require.Error(t, err)

		_, _, err = OpenCheckout(ctx, account, attemptID, price.Money{Amount: decimal.Zero, Currency: currency.EUR}, "https://ret", "https://wh")
		require.ErrorIs(t, err, ErrNonPositiveTotal)

		_, _, err = OpenCheckout(ctx, account, attemptID, price.Money{Amount: decimal.NewFromInt(10), Currency: currency.USD}, "https://ret", "https://wh")
		require.Error(t, err)

		// PSP outage & invalid PSP responses leave DB completely untouched
		for _, tc := range []struct {
			name   string
			status PaymentStatus
			err    error
		}{
			{"psp error", PaymentStatus{}, errors.New("psp offline")},
			{"missing ref", PaymentStatus{State: paystate.STATUS_PENDING, Amount: price.Money{Amount: decimal.RequireFromString("10.70"), Currency: currency.EUR}}, nil},
			{"unknown state", PaymentStatus{ProviderRef: "ref", State: "UNKNOWN", Amount: price.Money{Amount: decimal.RequireFromString("10.70"), Currency: currency.EUR}}, nil},
			{"wrong amount", PaymentStatus{ProviderRef: "ref", State: paystate.STATUS_PENDING, Amount: price.Money{Amount: decimal.NewFromInt(10), Currency: currency.EUR}}, nil},
			{"wrong currency", PaymentStatus{ProviderRef: "ref", State: paystate.STATUS_PENDING, Amount: price.Money{Amount: decimal.RequireFromString("10.70"), Currency: currency.USD}}, nil},
		} {
			t.Run(tc.name, func(t *testing.T) {
				psp.status = tc.status
				psp.err = tc.err
				_, _, err := OpenCheckout(ctx, account, attemptID, subtotal, "https://ret", "https://wh")
				require.Error(t, err)
			})
		}
		psp.err = nil

		countAfterFailures, err := queries.GetQuerySet(&Payment{}).Count()
		require.NoError(t, err)
		require.Equal(t, countBefore, countAfterFailures, "OpenCheckout must never write to DB")

		// Valid OpenCheckout + RecordPayment
		initialProviderFee := &price.Money{Amount: decimal.RequireFromString("0.35"), Currency: currency.USD}
		psp.createStatusFn = func(req CreatePaymentRequest) PaymentStatus {
			return PaymentStatus{
				ProviderRef: "psp-tx-1",
				State:       paystate.STATUS_PENDING,
				Amount:      req.Amount,
				ProviderFee: initialProviderFee,
			}
		}
		defer func() { psp.createStatusFn = nil }()

		quote, checkout, err := OpenCheckout(ctx, account, attemptID, subtotal, "https://ret", "https://wh")
		require.NoError(t, err)
		require.True(t, quote.Subtotal.Equal(decimal.NewFromInt(10)))
		require.True(t, quote.Fee.Equal(decimal.RequireFromString("0.70")))
		require.True(t, quote.Total.Equal(decimal.RequireFromString("10.70")))
		require.Equal(t, currency.EUR, quote.Currency)
		require.NotSame(t, initialProviderFee, checkout.ProviderFee)

		// Still zero DB writes before RecordPayment
		countBeforeRecord, err := queries.GetQuerySet(&Payment{}).Count()
		require.NoError(t, err)
		require.Equal(t, countBefore, countBeforeRecord)

		order := seedOrder()

		// Signal veto rolls back RecordPayment
		vetoRecv, err := Signals.Start.Listen(ctx, func(context.Context, signals.Signal[PaymentStartedSignal], PaymentStartedSignal) error {
			return errors.New("start veto")
		})
		require.NoError(t, err)
		_, err = RecordPayment(ctx, order, account, attemptID, quote, checkout)
		Signals.Start.Disconnect(ctx, vetoRecv)
		require.Error(t, err)
		_, err = getAttempt(ctx, attemptID)
		require.Error(t, err)

		startsBefore := starts
		payment, err := RecordPayment(ctx, order, account, attemptID, quote, checkout)
		require.NoError(t, err)
		require.Equal(t, startsBefore+1, starts)
		require.Equal(t, attemptID, payment.ID)
		require.Equal(t, "EUR", payment.Currency)
		require.Equal(t, currency.EUR, payment.CurrencyUnit())
		require.True(t, payment.Amount.Equal(decimal.RequireFromString("10.70")))
		require.True(t, payment.Fee.Equal(decimal.RequireFromString("0.70")))
		require.True(t, payment.RefundedAmount.IsZero())
		require.Equal(t, account.InternalName, payment.Provider)
		require.Equal(t, "psp-tx-1", payment.TransactionID.String)
		require.Len(t, payment.History.Data, 1)
		require.Equal(t, EventCheckoutOpened, payment.History.Data[0].Kind)
		require.NotNil(t, payment.ProviderFee.Data)
		require.True(t, payment.ProviderFee.Data.Amount.Equal(decimal.RequireFromString("0.35")))
	})

	t.Run("CreateFreePayment records zero-total order without contacting PSP", func(t *testing.T) {
		order := seedOrder()
		id := drivers.NewULID()

		_, err := CreateFreePayment(ctx, (*Order)(nil), currency.EUR, id)
		require.Error(t, err)
		_, err = CreateFreePayment(ctx, order, currency.EUR, drivers.ULID{})
		require.Error(t, err)
		_, err = CreateFreePayment(ctx, order, currency.Unit{}, id)
		require.Error(t, err)

		beforeCreates, beforeGets, beforeRefunds, beforeCancels := psp.creates, psp.gets, psp.refunds, psp.cancels
		p, err := CreateFreePayment(ctx, order, currency.EUR, id)
		require.NoError(t, err)
		require.True(t, p.IsFreeCheckout())
		require.Equal(t, "EUR", p.Currency)
		require.Equal(t, currency.EUR, p.CurrencyUnit())
		require.Len(t, p.History.Data, 1)
		require.Equal(t, EventNoPaymentRequired, p.History.Data[0].Kind)

		refreshed, err := Refresh(ctx, id)
		require.NoError(t, err)
		require.True(t, refreshed.IsFreeCheckout())

		_, err = Refund(ctx, id, "free-refund", decimal.NewFromInt(1))
		require.ErrorIs(t, err, ErrUnsupportedOperation)
		err = Cancel(ctx, id, "free-cancel")
		require.ErrorIs(t, err, ErrUnsupportedOperation)
		require.Equal(t, beforeCreates, psp.creates)
		require.Equal(t, beforeGets, psp.gets)
		require.Equal(t, beforeRefunds, psp.refunds)
		require.Equal(t, beforeCancels, psp.cancels)
	})

	t.Run("Refresh and ServeWebhook reconcile status and provider fees", func(t *testing.T) {
		p := seedPayment(paystate.STATUS_PENDING)
		psp.status = PaymentStatus{
			ProviderRef: p.TransactionID.String,
			State:       paystate.STATUS_PENDING,
			Amount:      price.Money{Amount: p.Amount, Currency: currency.EUR},
			ProviderFee: &price.Money{Amount: decimal.RequireFromString("0.25"), Currency: currency.USD},
		}

		// Fee-only reconciliation updates ProviderFee and History without emitting Signals.Update
		updatesBefore := updates
		got, err := Refresh(ctx, p.ID)
		require.NoError(t, err)
		require.Equal(t, updatesBefore, updates)
		require.NotNil(t, got.ProviderFee.Data)
		require.True(t, got.ProviderFee.Data.Amount.Equal(decimal.RequireFromString("0.25")))
		require.Len(t, got.History.Data, 1)
		require.Equal(t, EventProviderFeeUpdated, got.History.Data[0].Kind)

		// Invalid provider fee rejected
		psp.status.ProviderFee = &price.Money{Amount: decimal.NewFromInt(-1), Currency: currency.USD}
		_, err = Refresh(ctx, p.ID)
		require.Error(t, err)
		psp.status.ProviderFee = nil

		// Webhook HTTP cases and successful state transition to PAID
		psp.status.State = paystate.STATUS_PAID
		for _, tc := range []struct {
			name, acctID, body string
			err                error
			code               int
		}{
			{"invalid account ULID", "bad", "", nil, http.StatusNotFound},
			{"unknown account", drivers.NewULID().String(), "", nil, http.StatusNotFound},
			{"oversized body", account.ID.String(), strings.Repeat("x", (1<<20)+1), nil, http.StatusBadRequest},
			{"invalid signature", account.ID.String(), "", ErrInvalidWebhook, http.StatusBadRequest},
			{"provider error", account.ID.String(), "", errors.New("offline"), http.StatusServiceUnavailable},
			{"success", account.ID.String(), "", nil, http.StatusOK},
			{"replay", account.ID.String(), "", nil, http.StatusOK},
		} {
			t.Run(tc.name, func(t *testing.T) {
				psp.err = tc.err
				rec := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(tc.body))
				ServeWebhook(rec, req, tc.acctID)
				require.Equal(t, tc.code, rec.Code)
			})
		}
		psp.err = nil
		require.Equal(t, updatesBefore+1, updates, "webhook replay must not emit duplicate update signals")

		// Stale regression from PAID -> PENDING is ignored
		psp.status.State = paystate.STATUS_PENDING
		got, err = Refresh(ctx, p.ID)
		require.NoError(t, err)
		require.Equal(t, paystate.STATUS_PAID, got.State)
		// Previous ProviderFee preserved when omitted
		require.NotNil(t, got.ProviderFee.Data)
		require.True(t, got.ProviderFee.Data.Amount.Equal(decimal.RequireFromString("0.25")))
	})

	t.Run("Refund partial, full, replay, and validation", func(t *testing.T) {
		p := seedPayment(paystate.STATUS_PAID)

		// Validation errors do not call PSP
		before := psp.refunds
		for _, tc := range []struct {
			name string
			ctx  context.Context
			key  string
			amt  string
		}{
			{"empty key", ctx, "  ", "1"},
			{"long key", ctx, strings.Repeat("x", 129), "1"},
			{"zero amount", ctx, "zero", "0"},
			{"negative amount", ctx, "neg", "-1"},
			{"excess precision", ctx, "prec", "1.001"},
			{"exceeds total", ctx, "over", "10.01"},
			{"rollback-only ctx", queries.CommitContext(ctx, false), "rb", "1"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				_, err := Refund(tc.ctx, p.ID, tc.key, decimal.RequireFromString(tc.amt))
				require.Error(t, err)
				require.Equal(t, before, psp.refunds)
			})
		}

		// Nested transaction rejected
		txCtx, tx, err := queries.StartTransaction(ctx)
		require.NoError(t, err)
		_, err = Refund(txCtx, p.ID, "nested", decimal.NewFromInt(1))
		require.Error(t, err)
		_ = tx.Rollback(txCtx)

		// Partial refund + idempotent replay
		res, err := Refund(ctx, p.ID, "partial-1", decimal.NewFromInt(4))
		require.NoError(t, err)
		require.Equal(t, "refund-receipt", res.ProviderRef)
		require.Equal(t, before+1, psp.refunds)

		replayRes, err := Refund(ctx, p.ID, "partial-1", decimal.NewFromInt(4))
		require.NoError(t, err)
		require.Equal(t, res, replayRes)
		require.Equal(t, before+1, psp.refunds)

		// Reused requestID with different amount fails
		_, err = Refund(ctx, p.ID, "partial-1", decimal.NewFromInt(5))
		require.Error(t, err)

		// Over-refunding remaining balance fails
		_, err = Refund(ctx, p.ID, "too-much", decimal.NewFromInt(7))
		require.Error(t, err)

		saved, err := getAttempt(ctx, p.ID)
		require.NoError(t, err)
		require.Equal(t, paystate.STATUS_PARTIAL_REFUND, saved.State)
		require.True(t, saved.RefundedAmount.Equal(decimal.NewFromInt(4)))

		// Disabled account still permits refunding an existing paid order
		account.Enabled = false
		_, err = queries.GetQuerySet(&ProviderAccount{}).Filter("ID", account.ID).Select("Enabled").Update(account)
		require.NoError(t, err)
		defer func() {
			account.Enabled = true
			_, _ = queries.GetQuerySet(&ProviderAccount{}).Filter("ID", account.ID).Select("Enabled").Update(account)
		}()

		_, err = Refund(ctx, p.ID, "remaining", decimal.NewFromInt(6))
		require.NoError(t, err)

		saved, err = getAttempt(ctx, p.ID)
		require.NoError(t, err)
		require.Equal(t, paystate.STATUS_REFUND, saved.State)
		require.True(t, saved.RefundedAmount.Equal(decimal.NewFromInt(10)))
		require.Len(t, saved.History.Data, 2)
		require.Equal(t, EventRefunded, saved.History.Data[0].Kind)
		require.Equal(t, EventRefunded, saved.History.Data[1].Kind)
	})

	t.Run("Cancel and unsupported capabilities", func(t *testing.T) {
		p := seedPayment(paystate.STATUS_PENDING)
		before := psp.cancels
		require.NoError(t, Cancel(ctx, p.ID, "cancel-1"))
		require.Equal(t, before+1, psp.cancels)

		// Idempotent replay
		require.NoError(t, Cancel(ctx, p.ID, "cancel-1"))
		require.Equal(t, before+1, psp.cancels)

		saved, err := getAttempt(ctx, p.ID)
		require.NoError(t, err)
		require.Equal(t, paystate.STATUS_CANCEL, saved.State)
		require.Len(t, saved.History.Data, 1)
		require.Equal(t, EventCancelled, saved.History.Data[0].Kind)

		// Paid payment cannot be cancelled
		paid := seedPayment(paystate.STATUS_PAID)
		require.Error(t, Cancel(ctx, paid.ID, "cancel-paid"))

		// Provider without Refunder / Canceller returns ErrUnsupportedOperation
		basicAccount, err := queries.GetQuerySet(&ProviderAccount{}).Create(&ProviderAccount{
			InternalName: "basic-test",
			Enabled:      true,
		})
		require.NoError(t, err)
		basicPayment, err := queries.GetQuerySet(&Payment{}).Create(&Payment{
			Order:         seedOrder(),
			Account:       basicAccount,
			Provider:      basicAccount.InternalName,
			TransactionID: sql.NullString{String: drivers.NewULID().String(), Valid: true},
			Amount:        decimal.NewFromInt(10),
			Currency:      "EUR",
			State:         paystate.STATUS_PAID,
		})
		require.NoError(t, err)
		_, err = Refund(ctx, basicPayment.ID, "unsupported-refund", decimal.NewFromInt(1))
		require.ErrorIs(t, err, ErrUnsupportedOperation)
		require.ErrorIs(t, Cancel(ctx, basicPayment.ID, "unsupported-cancel"), ErrUnsupportedOperation)
	})

	require.Positive(t, refundSignals)
	require.Positive(t, cancelSignals)

	t.Run("provider sync", testProviderSync)
}
