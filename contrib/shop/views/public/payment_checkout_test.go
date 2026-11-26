//go:build test

package public_test

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/contrib/shop/carts"
	"github.com/Nigel2392/go-django/contrib/shop/coupons"
	"github.com/Nigel2392/go-django/contrib/shop/customers"
	"github.com/Nigel2392/go-django/contrib/shop/orders"
	"github.com/Nigel2392/go-django/contrib/shop/payments"
	"github.com/Nigel2392/go-django/contrib/shop/products"
	"github.com/Nigel2392/go-django/contrib/shop/util/paystate"
	"github.com/Nigel2392/go-django/contrib/shop/views/public"
	"github.com/Nigel2392/go-django/djester"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func testPaymentCheckout(d *djester.Tester, t *testing.T, product, otherProduct *products.Product) {
	ctx := t.Context()
	_, err := queries.GetQuerySet(&CartTestUser{}).ExplicitSave().Create(&CartTestUser{
		ID: 72, Name: "checkout_user", Email: "buyer@example.test", FirstName: "Test", LastName: "Buyer", LoggedIn: true,
	})
	require.NoError(t, err)
	_, err = queries.GetQuerySet(&CartTestUser{}).ExplicitSave().Create(&CartTestUser{
		ID: 73, Name: "other_checkout_user", Email: "other-buyer@example.test", LoggedIn: true,
	})
	require.NoError(t, err)
	previousRedirect := d.Client().CheckRedirect
	d.Client().CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	defer func() { d.Client().CheckRedirect = previousRedirect }()

	provider := &testPaymentProvider{}
	payments.RegisterProvider("checkout-endpoint", func(context.Context, *payments.ProviderAccount) (payments.Provider, error) {
		return provider, nil
	})
	accounts := queries.GetQuerySet(&payments.ProviderAccount{})
	account, err := accounts.Create(&payments.ProviderAccount{
		Driver: "checkout-endpoint", Label: "Hosted payment", Enabled: true, Currency: "EUR",
		FixedFee: decimal.RequireFromString("0.50"),
	})
	require.NoError(t, err)
	other, err := accounts.Create(&payments.ProviderAccount{
		Driver: "checkout-endpoint", Label: "Other payment", Enabled: true, Currency: "EUR",
	})
	require.NoError(t, err)
	disabled, err := accounts.ExplicitSave().Create(&payments.ProviderAccount{
		Driver: "checkout-endpoint", Label: "Disabled payment", Currency: "EUR",
	})
	require.NoError(t, err)
	checkoutPath := django.Reverse("api:payments:checkout")
	post := func(id string, want int) *http.Response {
		t.Helper()
		response, err := d.PostJson(checkoutPath, nil, nil, public.PaymentCheckoutRequest{AccountID: id}, nil)
		require.NoError(t, err)
		defer response.Body.Close()
		if response.StatusCode != want {
			var body struct{ Message string }
			_ = response.JSON(&body)
			t.Fatalf("checkout returned %d, want %d: %s", response.StatusCode, want, body.Message)
		}
		return response.Response
	}
	_, _ = d.Logout()
	post(account.ID.String(), http.StatusUnauthorized)
	login, err := d.Login("checkout_user")
	require.NoError(t, err)
	login.Body.Close()

	post(account.ID.String(), http.StatusNotFound)
	post("invalid", http.StatusBadRequest)
	post(drivers.NewULID().String(), http.StatusNotFound)
	post(disabled.ID.String(), http.StatusNotFound)
	validJSON := `{"account_id":"` + account.ID.String() + `"}`
	for _, tc := range []struct {
		contentType string
		body        string
		status      int
	}{
		{"application/json", validJSON + `{}`, http.StatusBadRequest},
		{"application/json", validJSON + strings.Repeat(" ", 8192), http.StatusRequestEntityTooLarge},
		{"text/plain", validJSON, http.StatusUnsupportedMediaType},
	} {
		response, err := d.Post(checkoutPath, http.Header{"Content-Type": {tc.contentType}}, nil, strings.NewReader(tc.body))
		require.NoError(t, err)
		response.Body.Close()
		require.Equal(t, tc.status, response.StatusCode)
	}
	require.Empty(t, provider.requests)
	cartPath := django.Reverse("api:carts")
	getCart := func() *carts.Cart {
		t.Helper()
		var body struct {
			Cart *carts.Cart `json:"cart"`
		}
		response, err := d.Get(cartPath, nil, nil)
		require.NoError(t, err)
		defer response.Body.Close()
		require.Equal(t, http.StatusOK, response.StatusCode)
		require.NoError(t, response.JSON(&body))
		require.NotNil(t, body.Cart)
		return body.Cart
	}
	cart := getCart()
	post(account.ID.String(), http.StatusConflict)
	_, err = queries.GetQuerySet(&carts.CartItem{}).Create(&carts.CartItem{Cart: cart, Product: product, Amount: 1})
	require.NoError(t, err)
	orderCount, err := queries.GetQuerySet(&orders.Order{}).Count()
	require.NoError(t, err)
	freeCoupon, err := queries.GetQuerySet(&coupons.Coupon{}).Create(&coupons.Coupon{
		Code: "FREE-CHECKOUT", Title: "Full checkout discount", MaxDiscount: decimal.NewFromInt(1000),
	})
	require.NoError(t, err)
	redemptions := queries.GetQuerySet(&coupons.Redemption{})
	freeRedemption, err := redemptions.Create(&coupons.Redemption{Cart: cart, Coupon: freeCoupon})
	require.NoError(t, err)
	post(account.ID.String(), http.StatusUnprocessableEntity)
	count, err := queries.GetQuerySet(&orders.Order{}).Count()
	require.NoError(t, err)
	require.Equal(t, orderCount, count)
	require.Equal(t, cart.ID, getCart().ID)
	_, err = redemptions.Filter("ID", freeRedemption.ID).Select("Invalidated").Update(&coupons.Redemption{Invalidated: true})
	require.NoError(t, err)

	coupon, err := queries.GetQuerySet(&coupons.Coupon{}).Create(&coupons.Coupon{
		Code: "CHECKOUT", Title: "Checkout discount", MaxDiscount: decimal.NewFromInt(5),
	})
	require.NoError(t, err)
	redemption, err := redemptions.Create(&coupons.Redemption{Cart: cart, Coupon: coupon})
	require.NoError(t, err)
	unusedCoupon, err := queries.GetQuerySet(&coupons.Coupon{}).Create(&coupons.Coupon{
		Code: "OTHER-CHECKOUT", Title: "Discount for a removed product", MaxDiscount: decimal.NewFromInt(3),
	})
	require.NoError(t, err)
	attrs.Define(ctx, unusedCoupon).Get("Products")
	_, err = unusedCoupon.Products.Objects().AddTargets(otherProduct)
	require.NoError(t, err)
	unusedRedemption, err := redemptions.Create(&coupons.Redemption{Cart: cart, Coupon: unusedCoupon})
	require.NoError(t, err)
	post(disabled.ID.String(), http.StatusNotFound)
	require.Empty(t, provider.requests)

	provider.startErr = errors.New("provider did not respond")
	post(account.ID.String(), http.StatusBadGateway)
	savedCart, err := queries.GetQuerySet(&carts.Cart{}).Filter("ID", cart.ID).Get()
	require.NoError(t, err)
	require.False(t, savedCart.Object.Active)
	attempt, err := queries.GetQuerySet(&payments.Payment{}).Filter("ID", cart.ID).Select("*", "Order.*").Get()
	require.NoError(t, err)
	require.False(t, attempt.Object.TransactionID.Valid)
	order := attempt.Object.Order.(*orders.Order)
	customer, err := queries.GetQuerySet(&customers.Customer{}).Filter("ID", order.Customer.ID).Get()
	require.NoError(t, err)
	require.Equal(t, "buyer@example.test", customer.Object.Email.Address)
	require.Equal(t, "Test", customer.Object.FirstName)
	require.Equal(t, "Buyer", customer.Object.LastName)
	line, err := queries.GetQuerySet(&orders.OrderItem{}).WithContext(ctx).Filter("Order", order.ID).Get()
	require.NoError(t, err)
	require.Equal(t, product.ID, line.Object.Product.Data.Product.ID)
	require.True(t, line.Object.TaxRate.Equal(product.TaxRate))
	redeemed, err := redemptions.Filter("ID", redemption.ID).Get()
	require.NoError(t, err)
	require.Equal(t, order.ID, redeemed.Object.Order.(*orders.Order).ID)
	require.Equal(t, customer.Object.ID, redeemed.Object.Customer.ID)
	require.False(t, redeemed.Object.Invalidated)
	unused, err := redemptions.Filter("ID", unusedRedemption.ID).Get()
	require.NoError(t, err)
	require.True(t, unused.Object.Invalidated)
	require.Nil(t, unused.Object.Order)

	newCart := getCart()
	require.NotEqual(t, cart.ID, newCart.ID)
	post(other.ID.String(), http.StatusConflict)
	provider.startErr = nil
	form := url.Values{"account_id": {account.ID.String()}}
	response, err := d.Post(checkoutPath, http.Header{
		"Content-Type":     {"application/x-www-form-urlencoded"},
		"X-Forwarded-Host": {"untrusted.example.test"},
	}, nil, strings.NewReader(form.Encode()))
	require.NoError(t, err)
	response.Body.Close()
	require.Equal(t, http.StatusSeeOther, response.StatusCode)
	require.Equal(t, "https://checkout.example.test/pay", response.Header.Get("Location"))
	replay := post(account.ID.String(), http.StatusSeeOther)
	require.Equal(t, response.Header.Get("Location"), replay.Header.Get("Location"))
	require.Len(t, provider.requests, 2)
	require.Equal(t, provider.requests[0], provider.requests[1])
	create := provider.requests[1]
	require.Equal(t, cart.ID, create.AttemptID)
	require.True(t, create.Amount.Amount.Equal(decimal.RequireFromString("30.74")), "got %s", create.Amount.Amount)
	require.Equal(t, "https://merchant.example.test/order/thanks?payment_id="+cart.ID.String(), create.ReturnURL)
	require.Equal(t, "https://merchant.example.test"+django.Reverse("api:payments:webhook", account.ID), create.WebhookURL)
	count, err = queries.GetQuerySet(&orders.Order{}).Filter("Customer", customer.Object.ID).Count()
	require.NoError(t, err)
	require.EqualValues(t, 1, count)

	provider.status.State = paystate.STATUS_PAID
	_, err = payments.Refresh(ctx, cart.ID)
	require.NoError(t, err)
	savedPayment, err := payments.Refresh(ctx, cart.ID)
	require.NoError(t, err)
	require.Len(t, savedPayment.History.Data, 4)
	require.Equal(t, payments.EventPrepared, savedPayment.History.Data[0].Kind)
	require.Equal(t, payments.EventCheckoutFailed, savedPayment.History.Data[1].Kind)
	require.Equal(t, payments.EventCheckoutOpened, savedPayment.History.Data[2].Kind)
	require.Equal(t, payments.EventStatusUpdated, savedPayment.History.Data[3].Kind)
	require.Equal(t, paystate.STATUS_PAID, savedPayment.History.Data[3].State)

	_, _ = d.Logout()
	login, err = d.Login("other_checkout_user")
	require.NoError(t, err)
	login.Body.Close()
	post(account.ID.String(), http.StatusNotFound)
	require.Len(t, provider.requests, 2)
}
