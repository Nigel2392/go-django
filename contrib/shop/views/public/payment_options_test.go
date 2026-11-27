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
	"github.com/Nigel2392/go-django/contrib/shop/carts"
	"github.com/Nigel2392/go-django/contrib/shop/coupons"
	"github.com/Nigel2392/go-django/contrib/shop/orders"
	"github.com/Nigel2392/go-django/contrib/shop/payments"
	"github.com/Nigel2392/go-django/contrib/shop/shared/price"
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
	"golang.org/x/text/currency"
)

type testPaymentProvider struct {
	requests  []payments.CreatePaymentRequest
	refunds   []payments.RefundRequest
	status    payments.PaymentStatus
	startErr  error
	refundErr error
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

func (p *testPaymentProvider) RefundPayment(ctx context.Context, req payments.RefundRequest) (payments.RefundResult, error) {
	if queries.IsTransactionContext(ctx) {
		return payments.RefundResult{}, errors.New("provider refund called inside a database transaction")
	}
	p.refunds = append(p.refunds, req)
	if p.refundErr != nil {
		return payments.RefundResult{}, p.refundErr
	}
	return payments.RefundResult{ProviderRef: "refund-" + req.RequestID}, nil
}

func testPaymentOptions(d *djester.Tester, t *testing.T) {
	ctx := t.Context()
	provider := &testPaymentProvider{}
	payments.RegisterProvider(&payments.PSP{
		Name:              "checkout-test",
		TextLabel:         func(context.Context) string { return "Test checkout" },
		CurrencyUnit:      currency.EUR,
		FixedFeeAmount:    decimal.RequireFromString("0.25"),
		PercentageFeeRate: decimal.NewFromInt(2),
		Factory: func(context.Context, *payments.ProviderAccount) (payments.Provider, error) {
			return provider, nil
		},
	})
	payments.RegisterProvider(&payments.PSP{
		Name:           "checkout-test-alt",
		TextLabel:      func(context.Context) string { return "Alternative checkout" },
		CurrencyUnit:   currency.EUR,
		FixedFeeAmount: decimal.RequireFromString("0.50"),
		Factory: func(context.Context, *payments.ProviderAccount) (payments.Provider, error) {
			return provider, nil
		},
	})
	account, err := queries.GetQuerySet(&payments.ProviderAccount{}).Create(&payments.ProviderAccount{
		InternalName: "checkout-test",
		Enabled:      true,
	})
	if err != nil {
		t.Fatal(err)
	}
	other, err := queries.GetQuerySet(&payments.ProviderAccount{}).Create(&payments.ProviderAccount{
		InternalName: "checkout-test-alt",
		Enabled:      true,
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
	if !options[0].Fee.Equal(decimal.RequireFromString("0.50")) || options[0].Currency != currency.EUR {
		t.Fatalf("unexpected alternative quote: %+v", options[0])
	}
	option := options[1]
	if !option.Subtotal.Equal(decimal.RequireFromString("24.19")) ||
		!option.Fee.Equal(decimal.RequireFromString("0.73")) ||
		!option.Total.Equal(decimal.RequireFromString("24.92")) ||
		option.Currency != currency.EUR {
		t.Fatalf("unexpected payment quote: %+v", option)
	}

	var activeCart public.CartResponse
	customerResp, err := d.PostJson(django.Reverse("api:carts:customer"), nil, nil, public.CartCustomerRequest{
		Email:     "checkout@example.test",
		FirstName: "Test",
		LastName:  "Customer",
	}, &activeCart)
	require.NoError(t, err)
	defer customerResp.Body.Close()
	require.Equal(t, http.StatusOK, customerResp.StatusCode)
	require.NotNil(t, activeCart.Cart)
	require.NotNil(t, activeCart.Cart.Customer)
	customer := activeCart.Cart.Customer

	req := httptest.NewRequest(http.MethodPost, "/checkout", nil).WithContext(ctx)
	const returnURL = "https://merchant.example.test/return"
	webhookPath := django.Reverse("api:payments:webhook", account.ID)
	webhookURL := "https://merchant.example.test" + webhookPath
	baseOpts := orders.CheckoutOptions{
		Cart:      activeCart.Cart,
		AccountID: account.ID,
		Currency:  currency.EUR,
		ReturnURL: func(paymentID, _ drivers.ULID) string {
			return returnURL + "?payment_id=" + paymentID.String()
		},
		WebhookURL: func(*payments.ProviderAccount) string {
			return webhookURL
		},
	}

	t.Run("invalid account leaves cart active with zero orders and payments", func(t *testing.T) {
		bad := baseOpts
		bad.AccountID = drivers.NewULID()
		_, err := orders.CheckoutCart(req, bad)
		require.ErrorIs(t, err, orders.ErrCheckoutAccountUnavailable)
		require.ErrorIs(t, err, dberrors.NoRows)
		for _, count := range []func() (int64, error){
			queries.GetQuerySet(&orders.Order{}).Filter("Customer", customer.ID).Count,
			queries.GetQuerySet(&orders.OrderItem{}).Count,
			queries.GetQuerySet(&payments.Payment{}).Count,
		} {
			n, err := count()
			require.NoError(t, err)
			require.Zero(t, n)
		}
		persistedCart, err := queries.GetQuerySet(&carts.Cart{}).Filter("ID", activeCart.Cart.ID).Get()
		require.NoError(t, err)
		require.True(t, persistedCart.Object.Active)
		require.Empty(t, provider.requests)
	})

	t.Run("custom pricer and PSP error leave cart active with zero orders and payments", func(t *testing.T) {
		lostResponse := errors.New("provider response lost")
		provider.startErr = lostResponse
		defer func() { provider.startErr = nil }()

		custom := baseOpts
		custom.Pricer = func(ctx context.Context, cart *carts.CartRow) (price.Money, error) {
			require.False(t, queries.IsTransactionContext(ctx))
			require.Len(t, cart.Items, 1)
			require.False(t, cart.Items[0].CartItem.Product.ID.IsZero())
			require.NotEmpty(t, cart.Items[0].CartItem.Product.Title)
			require.True(t, cart.Totals.Payable().Round(2).Equal(option.Subtotal))
			return price.Money{Amount: decimal.NewFromInt(31), Currency: currency.EUR}, nil
		}

		_, err := orders.CheckoutCart(req, custom)
		require.ErrorIs(t, err, lostResponse)
		require.Len(t, provider.requests, 1)
		require.True(t, provider.requests[0].Amount.Amount.Equal(decimal.RequireFromString("31.87")))
		require.Equal(t, currency.EUR, provider.requests[0].Amount.Currency)
		provider.requests = nil

		for _, count := range []func() (int64, error){
			queries.GetQuerySet(&orders.Order{}).Filter("Customer", customer.ID).Count,
			queries.GetQuerySet(&orders.OrderItem{}).Count,
			queries.GetQuerySet(&payments.Payment{}).Count,
		} {
			n, err := count()
			require.NoError(t, err)
			require.Zero(t, n)
		}
		persistedCart, err := queries.GetQuerySet(&carts.Cart{}).Filter("ID", activeCart.Cart.ID).Get()
		require.NoError(t, err)
		require.True(t, persistedCart.Object.Active)
	})

	result, err := orders.CheckoutCart(req, baseOpts)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, result.Free)
	order, attempt := result.Order, result.Payment
	require.NotNil(t, order)
	require.NotNil(t, attempt)
	require.Equal(t, paystate.STATUS_PENDING, attempt.State)
	require.True(t, attempt.TransactionID.Valid)
	require.Equal(t, "test-reference", attempt.TransactionID.String)
	require.Equal(t, "https://checkout.example.test/pay", result.Checkout.URL)
	require.Len(t, provider.requests, 1)
	require.Equal(t, attempt.ID, provider.requests[0].AttemptID)
	if !attempt.Fee.Equal(option.Fee) || !attempt.Amount.Equal(option.Total) {
		t.Fatalf("checkout amount differs from cart quote: %+v", attempt)
	}
	persistedCart, err := queries.GetQuerySet(&carts.Cart{}).Filter("ID", activeCart.Cart.ID).Get()
	require.NoError(t, err)
	require.False(t, persistedCart.Object.Active)

	line, err := queries.GetQuerySet(&orders.OrderItem{}).Filter("Order", order.ID).Get()
	require.NoError(t, err)
	require.NotEmpty(t, line.Object.Product.Data.Product.Title)
	require.IsType(t, &testCartSupplierProduct{}, line.Object.Product.Data.Specific)
	require.NotNil(t, line.Object.Pricing.Data)
	require.Len(t, line.Object.Pricing.Data.RedemptionAmounts, 2)

	t.Run("re-checking out an already converted cart returns ErrCheckoutConflict", func(t *testing.T) {
		_, err := orders.CheckoutCart(req, baseOpts)
		require.ErrorIs(t, err, orders.ErrCheckoutConflict)
		n, err := queries.GetQuerySet(&orders.Order{}).Filter("Customer", customer.ID).Count()
		require.NoError(t, err)
		require.EqualValues(t, 1, n)
		require.Len(t, provider.requests, 1)
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

	refundAmt := decimal.RequireFromString("5.00")
	refundRes, err := payments.Refund(ctx, attempt.ID, "refund-demo", refundAmt)
	require.NoError(t, err)
	require.Equal(t, "refund-refund-demo", refundRes.ProviderRef)
	require.Len(t, provider.refunds, 1)
	require.Equal(t, "test-reference", provider.refunds[0].ProviderRef)
	require.Equal(t, "refund-demo", provider.refunds[0].RequestID)
	require.True(t, provider.refunds[0].Amount.Amount.Equal(refundAmt))
	require.Equal(t, currency.EUR, provider.refunds[0].Amount.Currency)

	t.Run("audit committed checkout, reconciliation, and refund", func(t *testing.T) {
		lookup := func(kind, objectID string) *auditlogs.Entry {
			t.Helper()
			rows, err := queries.GetQuerySet(&auditlogs.Entry{}).Filter("Type", kind).OrderBy("-ID").All()
			require.NoError(t, err)
			for _, entry := range rows {
				if attrs.ToString(entry.Object.ObjectID()) == objectID {
					return entry.Object
				}
			}
			t.Fatalf("audit entry %s for object %s not found", kind, objectID)
			return nil
		}
		created := lookup("shop:orders:create", order.ID.String())
		require.Equal(t, logger.INF, created.Level())
		started := lookup("shop:payments:start", attempt.ID.String())
		require.Equal(t, order.ID.String(), started.Data()["order_id"])
		require.Equal(t, paystate.STATUS_PENDING, started.Data()["state"])
		updatedEntry := lookup("shop:payments:update", attempt.ID.String())
		require.Equal(t, paystate.STATUS_PARTIAL_REFUND, updatedEntry.Data()["state"])
		require.Equal(t, paystate.STATUS_PAID, updatedEntry.Data()["previous_state"])
		refunded := lookup("shop:payments:refund", attempt.ID.String())
		require.Equal(t, logger.INF, refunded.Level())
		require.Equal(t, "refund-demo", refunded.Data()["request_id"])
		require.Equal(t, "5", refunded.Data()["refund_amount"])
	})

	t.Run("clearing the source cart preserves order redemptions", func(t *testing.T) {
		response, err := d.PostJson(django.Reverse("api:carts:clear"), nil, nil, map[string]any{}, nil)
		require.NoError(t, err)
		response.Body.Close()
		require.Equal(t, http.StatusOK, response.StatusCode)

		for id, amount := range line.Object.Pricing.Data.RedemptionAmounts {
			redemption, err := queries.GetQuerySet(&coupons.Redemption{}).Filter("ID", id).Get()
			require.NoError(t, err)
			require.Equal(t, order.ID, redemption.Object.Order.(*orders.Order).ID)
			require.True(t, amount.Equal(redemption.Object.DiscountAmount))
			require.False(t, redemption.Object.Invalidated)
		}
	})
}
