//go:build test

package public_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	auditlogs "github.com/Nigel2392/go-django/contrib/reports/audit_logs"
	"github.com/Nigel2392/go-django/contrib/shop/broker"
	"github.com/Nigel2392/go-django/contrib/shop/carts"
	"github.com/Nigel2392/go-django/contrib/shop/customers"
	"github.com/Nigel2392/go-django/contrib/shop/orders"
	"github.com/Nigel2392/go-django/contrib/shop/payments"
	"github.com/Nigel2392/go-django/contrib/shop/util/paystate"
	"github.com/Nigel2392/go-django/contrib/shop/views/public"
	"github.com/Nigel2392/go-django/djester"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	dberrors "github.com/Nigel2392/go-django/queries/src/drivers/errors"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

type testPaymentProvider struct {
	requests []payments.CreatePaymentRequest
	status   payments.PaymentStatus
	startErr error
}

func (p *testPaymentProvider) CreatePayment(ctx context.Context, req payments.CreatePaymentRequest) (payments.Checkout, error) {
	if queries.IsTransactionContext(ctx) {
		return payments.Checkout{}, errors.New("provider called inside a database transaction")
	}
	p.requests = append(p.requests, req)
	p.status = payments.PaymentStatus{ProviderRef: "test-reference", State: paystate.STATUS_PENDING, Amount: req.Amount}
	if p.startErr != nil {
		return payments.Checkout{}, p.startErr
	}
	return payments.Checkout{PaymentStatus: p.status, URL: "https://checkout.example.test/pay"}, nil
}

func (p *testPaymentProvider) GetPayment(context.Context, string) (payments.PaymentStatus, error) {
	return p.status, nil
}

func (p *testPaymentProvider) HandleWebhook(_ context.Context, req payments.WebhookRequest) (payments.WebhookResult, error) {
	if string(req.Body) != "provider callback" {
		return payments.WebhookResult{}, payments.ErrInvalidWebhook
	}
	return payments.WebhookResult{Payment: p.status}, nil
}

type checkoutPricerFunc func(context.Context, *carts.CartRow) (payments.Money, error)

func (fn checkoutPricerFunc) CheckoutSubtotal(ctx context.Context, cart *carts.CartRow) (payments.Money, error) {
	return fn(ctx, cart)
}

func testPaymentOptions(d *djester.Tester, t *testing.T) {
	ctx := t.Context()
	provider := &testPaymentProvider{}
	payments.RegisterProvider("checkout-test", func(context.Context, *payments.ProviderAccount) (payments.Provider, error) {
		return provider, nil
	})
	account, err := queries.GetQuerySet(&payments.ProviderAccount{}).Create(&payments.ProviderAccount{
		Driver: "checkout-test", Label: "Test checkout", Enabled: true, Currency: "EUR",
		FixedFee: decimal.RequireFromString("0.25"), PercentageFee: decimal.NewFromInt(2),
		CredentialRef: "test-secret-ref",
	})
	if err != nil {
		t.Fatal(err)
	}
	other, err := queries.GetQuerySet(&payments.ProviderAccount{}).Create(&payments.ProviderAccount{
		Driver: "checkout-test", Label: "Alternative checkout", Enabled: true, Currency: "EUR",
		FixedFee: decimal.RequireFromString("0.50"),
	})
	if err != nil {
		t.Fatal(err)
	}

	optionsURL := django.Reverse("api:payments:options")
	resp, err := d.Get(optionsURL, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("payment options returned %d", resp.StatusCode)
	}
	var options []public.PaymentOption
	if err := resp.JSON(&options); err != nil {
		t.Fatal(err)
	}
	if len(options) != 2 || options[1].AccountID != account.ID || options[0].AccountID != other.ID {
		t.Fatalf("unexpected payment options: %+v", options)
	}
	if !options[0].Fee.Equal(decimal.RequireFromString("0.50")) {
		t.Fatalf("unexpected alternative fee: %s", options[0].Fee)
	}
	option := options[1]
	if !option.Subtotal.Equal(decimal.RequireFromString("24.19")) ||
		!option.Fee.Equal(decimal.RequireFromString("0.73")) ||
		!option.Total.Equal(decimal.RequireFromString("24.92")) {
		t.Fatalf("unexpected payment quote: %+v", option)
	}

	customer, err := queries.GetQuerySet(&customers.Customer{}).Create(&customers.Customer{
		Email: drivers.MustParseEmail("checkout@example.test"), FirstName: "Test", LastName: "Customer",
	})
	if err != nil {
		t.Fatal(err)
	}
	cartResponse, err := d.Get(django.Reverse("api:carts"), nil, nil)
	require.NoError(t, err)
	defer cartResponse.Body.Close()
	require.Equal(t, http.StatusOK, cartResponse.StatusCode)
	var activeCart struct {
		Cart *carts.Cart `json:"cart"`
	}
	require.NoError(t, cartResponse.JSON(&activeCart))
	require.NotNil(t, activeCart.Cart)

	flow := broker.Broker{Currency: "EUR"}
	req := httptest.NewRequest(http.MethodPost, "/checkout", nil).WithContext(ctx)
	selection := broker.CheckoutRequest{
		Customer: customer, CartID: activeCart.Cart.ID, AccountID: account.ID, AttemptID: drivers.NewULID(),
	}
	const returnURL = "https://merchant.example.test/return"
	webhookPath := django.Reverse("api:payments:webhook", account.ID)
	webhookURL := "https://merchant.example.test" + webhookPath

	t.Run("invalid account rolls back the order and its lines", func(t *testing.T) {
		bad := selection
		bad.AccountID = drivers.NewULID()
		_, err := flow.PrepareCheckout(req, bad)
		require.ErrorIs(t, err, dberrors.NoRows)
		for _, count := range []func() (int64, error){
			queries.GetQuerySet(&orders.Order{}).Filter("Customer", customer.ID).Count,
			queries.GetQuerySet(&orders.OrderItem{}).Count,
			queries.GetQuerySet(&payments.Payment{}).Filter("ID", selection.AttemptID).Count,
		} {
			n, err := count()
			require.NoError(t, err)
			require.Zero(t, n)
		}
		require.Empty(t, provider.requests)
	})

	t.Run("custom pricing participates in the caller transaction", func(t *testing.T) {
		ctx, tx, err := queries.StartTransaction(t.Context())
		require.NoError(t, err)
		defer tx.Rollback(ctx)

		custom := broker.Broker{Pricer: checkoutPricerFunc(func(ctx context.Context, cart *carts.CartRow) (payments.Money, error) {
			require.True(t, queries.IsTransactionContext(ctx))
			require.Len(t, cart.Items, 1)
			require.False(t, cart.Items[0].CartItem.Product.ID.IsZero())
			require.NotEmpty(t, cart.Items[0].CartItem.Product.Title)
			require.True(t, cart.Totals.Payable().Round(2).Equal(option.Subtotal))
			return payments.Money{Amount: decimal.NewFromInt(31), Currency: "EUR"}, nil
		})}
		prepared, err := custom.PrepareCheckout(req.WithContext(ctx), selection)
		require.NoError(t, err)
		require.True(t, prepared.Payment.Amount.Equal(decimal.RequireFromString("31.87")))
		_, err = custom.StartCheckout(ctx, customer.ID, selection.AttemptID, returnURL, webhookURL)
		require.ErrorContains(t, err, "committed payment attempt")
		require.Empty(t, provider.requests)
		require.NoError(t, tx.Rollback(ctx))
		_, err = queries.GetQuerySet(&orders.Order{}).Filter("ID", prepared.Order.ID).Get()
		require.ErrorIs(t, err, dberrors.NoRows)
		_, err = queries.GetQuerySet(&payments.Payment{}).Filter("ID", prepared.Payment.ID).Get()
		require.ErrorIs(t, err, dberrors.NoRows)
	})

	prepared, err := flow.PrepareCheckout(req, selection)
	require.NoError(t, err)
	order, attempt := prepared.Order, prepared.Payment
	require.Equal(t, selection.AttemptID, attempt.ID)
	require.Equal(t, paystate.STATUS_INIT, attempt.State)
	require.Empty(t, provider.requests)
	if !attempt.Fee.Equal(option.Fee) || !attempt.Amount.Equal(option.Total) {
		t.Fatalf("checkout amount differs from cart quote: %+v", attempt)
	}
	line, err := queries.GetQuerySet(&orders.OrderItem{}).Filter("Order", order.ID).Get()
	require.NoError(t, err)
	require.NotEmpty(t, line.Object.Product.Data.Product.Title)
	require.IsType(t, &testCartSupplierProduct{}, line.Object.Product.Data.Specific)

	t.Run("retry reuses the saved order and prevents cross-customer access", func(t *testing.T) {
		retry := flow
		retry.Pricer = checkoutPricerFunc(func(context.Context, *carts.CartRow) (payments.Money, error) {
			return payments.Money{}, errors.New("saved attempt should skip repricing")
		})
		replayed, err := retry.PrepareCheckout(req, selection)
		require.NoError(t, err)
		require.Equal(t, order.ID, replayed.Order.ID)
		require.Equal(t, attempt.ID, replayed.Payment.ID)
		n, err := queries.GetQuerySet(&orders.Order{}).Filter("Customer", customer.ID).Count()
		require.NoError(t, err)
		require.EqualValues(t, 1, n)

		changed := selection
		changed.AccountID = other.ID
		_, err = flow.PrepareCheckout(req, changed)
		require.ErrorIs(t, err, broker.ErrCheckoutConflict)
		changed = selection
		changed.Customer = &customers.Customer{ID: drivers.NewULID()}
		_, err = flow.PrepareCheckout(req, changed)
		require.ErrorIs(t, err, broker.ErrCheckoutConflict)
		_, err = flow.StartCheckout(ctx, changed.Customer.ID, attempt.ID, returnURL, webhookURL)
		require.ErrorIs(t, err, dberrors.NoRows)
		require.Empty(t, provider.requests)
	})

	t.Run("retry after provider error uses the same payment attempt", func(t *testing.T) {
		lostResponse := errors.New("provider response lost")
		provider.startErr = lostResponse
		_, err := flow.StartCheckout(ctx, customer.ID, attempt.ID, returnURL, webhookURL)
		require.ErrorIs(t, err, lostResponse)
		saved, err := queries.GetQuerySet(&payments.Payment{}).Filter("ID", attempt.ID).Get()
		require.NoError(t, err)
		require.False(t, saved.Object.TransactionID.Valid)
		provider.startErr = nil
		for range 2 {
			checkout, err := flow.StartCheckout(ctx, customer.ID, attempt.ID, returnURL, webhookURL)
			require.NoError(t, err)
			require.NotEmpty(t, checkout.URL)
		}
		require.Len(t, provider.requests, 2)
		require.Equal(t, provider.requests[0], provider.requests[1])
		require.Equal(t, attempt.ID, provider.requests[0].AttemptID)
	})

	account.Enabled = false
	if _, err := queries.GetQuerySet(&payments.ProviderAccount{}).Filter("ID", account.ID).Select("Enabled").Update(account); err != nil {
		t.Fatal(err)
	}
	provider.status.State = paystate.STATUS_PAID
	provider.status.Amount.Amount = option.Total.Add(decimal.NewFromInt(1))
	post := func(body string, want int) {
		t.Helper()
		resp, err := d.Post(webhookPath, nil, nil, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != want {
			t.Fatalf("webhook returned %d, want %d", resp.StatusCode, want)
		}
	}
	post("invalid", http.StatusBadRequest)
	post("provider callback", http.StatusServiceUnavailable)
	provider.status.Amount.Amount = option.Total
	post("provider callback", http.StatusOK)
	post("provider callback", http.StatusOK)

	provider.status.State = paystate.STATUS_PENDING
	updated, err := payments.Refresh(ctx, attempt.ID)
	if err != nil || updated.State != paystate.STATUS_PAID {
		t.Fatalf("stale provider state overwrote paid: %+v, error %v", updated, err)
	}
	row, err := queries.GetQuerySet(&orders.Order{}).Preload(queries.NoJoins("Payments")).Filter("ID", order.ID).Get()
	if err != nil {
		t.Fatal(err)
	}
	history := row.Object.Payments.AsList()
	require.Equal(t, order.State, row.Object.State)
	if len(history) != 1 || history[0].ID != attempt.ID || history[0].State != paystate.STATUS_PAID {
		t.Fatalf("unexpected order payment history: %+v", history)
	}

	t.Run("audit committed checkout and reconciliation", func(t *testing.T) {
		lookup := func(kind, objectID string) *auditlogs.Entry {
			t.Helper()
			entry, err := queries.GetQuerySet(&auditlogs.Entry{}).Filter("Type", kind).Get()
			require.NoError(t, err)
			require.Equal(t, objectID, attrs.ToString(entry.Object.ObjectID()))
			return entry.Object
		}
		created := lookup("shop:orders:create", order.ID.String())
		require.Equal(t, logger.INF, created.Level())
		started := lookup("shop:payments:start", attempt.ID.String())
		require.Equal(t, order.ID.String(), started.Data()["order_id"])
		require.Equal(t, paystate.STATUS_PENDING, started.Data()["state"])
		paid := lookup("shop:payments:update", attempt.ID.String())
		require.Equal(t, paystate.STATUS_PAID, paid.Data()["state"])
		require.Equal(t, paystate.STATUS_PENDING, paid.Data()["previous_state"])
		operation := payments.PaymentOperationSignal{
			Payment: updated, RequestID: "refund-demo",
			Operation: payments.OperationRecord{Kind: payments.OperationRefund, Phase: payments.OperationUncertain},
			Error:     errors.New("raw provider response"),
		}
		require.NoError(t, payments.Signals.OperationFailed.Send(t.Context(), operation))
		failed := lookup("shop:payments:operation:failed", attempt.ID.String())
		require.Equal(t, logger.WRN, failed.Level())
		require.Equal(t, string(payments.OperationUncertain), failed.Data()["phase"])
		require.NotContains(t, failed.Data(), "error")
		req := httptest.NewRequest(http.MethodGet, "/audit", nil)
		require.Equal(t, "Payment operation needs reconciliation", auditlogs.Define(req, failed).Label())
	})
}
