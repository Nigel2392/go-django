//go:build test

package public_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/contrib/shop/customers"
	"github.com/Nigel2392/go-django/contrib/shop/orders"
	"github.com/Nigel2392/go-django/contrib/shop/payments"
	"github.com/Nigel2392/go-django/contrib/shop/util/paystate"
	"github.com/Nigel2392/go-django/contrib/shop/views/public"
	"github.com/Nigel2392/go-django/djester"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	django "github.com/Nigel2392/go-django/src"
	"github.com/shopspring/decimal"
)

type testPaymentProvider struct {
	requests int
	status   payments.PaymentStatus
}

func (p *testPaymentProvider) CreatePayment(_ context.Context, req payments.CreatePaymentRequest) (payments.Checkout, error) {
	p.requests++
	p.status = payments.PaymentStatus{ProviderRef: "test-reference", State: paystate.STATUS_PENDING, Amount: req.Amount}
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

	optionsURL := django.Reverse("api:carts:payment_options")
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
	order, err := queries.GetQuerySet(&orders.Order{}).Create(&orders.Order{Customer: customer})
	if err != nil {
		t.Fatal(err)
	}
	attempt, err := payments.CreateAttempt(ctx, order, account.ID, payments.Money{
		Amount: option.Subtotal, Currency: option.Currency,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !attempt.Fee.Equal(option.Fee) || !attempt.Amount.Equal(option.Total) {
		t.Fatalf("checkout amount differs from cart quote: %+v", attempt)
	}
	const returnURL = "https://merchant.example.test/return"
	webhookPath := django.Reverse("api:payment_webhook", account.ID)
	webhookURL := "https://merchant.example.test" + webhookPath
	for range 2 {
		checkout, err := payments.Start(ctx, attempt.ID, returnURL, webhookURL)
		if err != nil || checkout.URL == "" {
			t.Fatalf("checkout %+v, error %v", checkout, err)
		}
	}
	if provider.requests != 1 {
		t.Fatalf("expected one checkout request, got %d", provider.requests)
	}

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
	if len(history) != 1 || history[0].ID != attempt.ID || history[0].State != paystate.STATUS_PAID {
		t.Fatalf("unexpected order payment history: %+v", history)
	}
}
