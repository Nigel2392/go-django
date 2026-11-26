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
	redemptions := queries.GetQuerySet(&coupons.Redemption{})

	coupon, err := queries.GetQuerySet(&coupons.Coupon{}).Create(&coupons.Coupon{
		Code: "CHECKOUT", Title: "Checkout discount", MaxDiscount: decimal.RequireFromString("2.25"),
	})
	require.NoError(t, err)
	redemption, err := redemptions.Create(&coupons.Redemption{Cart: cart, Coupon: coupon})
	require.NoError(t, err)
	secondCoupon, err := queries.GetQuerySet(&coupons.Coupon{}).Create(&coupons.Coupon{
		Code: "SECOND-CHECKOUT", Title: "Another checkout discount", MaxDiscount: decimal.RequireFromString("2.75"),
	})
	require.NoError(t, err)
	secondRedemption, err := redemptions.Create(&coupons.Redemption{Cart: cart, Coupon: secondCoupon})
	require.NoError(t, err)
	zeroCoupon, err := queries.GetQuerySet(&coupons.Coupon{}).Create(&coupons.Coupon{
		Code: "ZERO-CHECKOUT", Title: "Zero checkout discount",
	})
	require.NoError(t, err)
	zeroRedemption, err := redemptions.Create(&coupons.Redemption{Cart: cart, Coupon: zeroCoupon})
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
	require.True(t, attempt.Object.Fee.Equal(decimal.RequireFromString("0.50")))
	require.True(t, attempt.Object.Amount.Sub(attempt.Object.Fee).Equal(decimal.RequireFromString("30.24")))
	require.Nil(t, attempt.Object.ProviderFee.Data)
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
	pricing := line.Object.Pricing.Data
	require.NotNil(t, pricing)
	require.True(t, pricing.Totals.TotalDiscount.Equal(decimal.NewFromInt(5)))
	require.Len(t, pricing.RedemptionAmounts, 2)
	for id, amount := range map[uint64]decimal.Decimal{
		redemption.ID:       decimal.RequireFromString("2.25"),
		secondRedemption.ID: decimal.RequireFromString("2.75"),
	} {
		redeemed, err := redemptions.Filter("ID", id).Get()
		require.NoError(t, err)
		require.Equal(t, order.ID, redeemed.Object.Order.(*orders.Order).ID)
		require.Equal(t, customer.Object.ID, redeemed.Object.Customer.ID)
		require.False(t, redeemed.Object.Invalidated)
		require.True(t, redeemed.Object.DiscountAmount.Equal(amount))
		require.True(t, pricing.RedemptionAmounts[id].Equal(amount))
	}
	for _, id := range []uint64{unusedRedemption.ID, zeroRedemption.ID} {
		unused, err := redemptions.Filter("ID", id).Get()
		require.NoError(t, err)
		require.True(t, unused.Object.Invalidated)
		require.Nil(t, unused.Object.Order)
	}

	newCart := getCart()
	require.NotEqual(t, cart.ID, newCart.ID)
	post(other.ID.String(), http.StatusConflict)
	// A changed account fee applies to new attempts. This attempt keeps its quote.
	_, err = accounts.Filter("ID", account.ID).Select("FixedFee").Update(&payments.ProviderAccount{
		FixedFee: decimal.NewFromInt(1),
	})
	require.NoError(t, err)
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
	count, err := queries.GetQuerySet(&orders.Order{}).Filter("Customer", customer.Object.ID).Count()
	require.NoError(t, err)
	require.EqualValues(t, 1, count)

	provider.status.State = paystate.STATUS_PAID
	provider.status.ProviderFee = &payments.Money{
		Amount:   decimal.RequireFromString("0.28"),
		Currency: "USD",
	}
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
	require.True(t, savedPayment.Fee.Equal(decimal.RequireFromString("0.50")))
	require.True(t, savedPayment.Amount.Equal(create.Amount.Amount))
	require.NotNil(t, savedPayment.ProviderFee.Data)
	require.Equal(t, "USD", savedPayment.ProviderFee.Data.Currency)
	require.True(t, savedPayment.ProviderFee.Data.Amount.Equal(decimal.RequireFromString("0.28")))

	_, _ = d.Logout()
	login, err = d.Login("other_checkout_user")
	require.NoError(t, err)
	login.Body.Close()
	post(account.ID.String(), http.StatusNotFound)
	require.Len(t, provider.requests, 2)

	freeCart := getCart()
	_, err = queries.GetQuerySet(&carts.CartItem{}).Create(&carts.CartItem{
		Cart:    freeCart,
		Product: product,
		Amount:  1,
	})
	require.NoError(t, err)
	freeCoupon, err := queries.GetQuerySet(&coupons.Coupon{}).Create(&coupons.Coupon{
		Code:         "FREE-CHECKOUT",
		Title:        "Full checkout discount",
		IsPercentage: true,
		MaxDiscount:  decimal.NewFromInt(100),
	})
	require.NoError(t, err)
	freeRedemption, err := redemptions.Create(&coupons.Redemption{
		Cart:   freeCart,
		Coupon: freeCoupon,
	})
	require.NoError(t, err)
	optionsResponse, err := d.Get(django.Reverse("api:payments:options"), nil, nil)
	require.NoError(t, err)
	defer optionsResponse.Body.Close()
	require.Equal(t, http.StatusOK, optionsResponse.StatusCode)
	var options []public.PaymentOption
	require.NoError(t, optionsResponse.JSON(&options))
	require.Len(t, options, 1)
	require.True(t, options[0].NoPaymentRequired)
	require.True(t, options[0].AccountID.IsZero())
	require.True(t, options[0].Total.IsZero())
	require.True(t, options[0].Fee.IsZero())

	freeCheckout := post("", http.StatusSeeOther)
	returnLocation, err := url.Parse(freeCheckout.Header.Get("Location"))
	require.NoError(t, err)
	require.Equal(t, "merchant.example.test", returnLocation.Host)
	require.Equal(t, "/order/thanks", returnLocation.Path)
	require.Equal(t, freeCart.ID.String(), returnLocation.Query().Get("payment_id"))

	freeAttempt, err := queries.GetQuerySet(&payments.Payment{}).
		Filter("ID", freeCart.ID).
		Select("*", "Order.*").
		Get()
	require.NoError(t, err)
	require.True(t, freeAttempt.Object.IsFreeCheckout())
	require.Nil(t, freeAttempt.Object.Account)
	require.False(t, freeAttempt.Object.TransactionID.Valid)
	require.True(t, freeAttempt.Object.Amount.IsZero())
	require.True(t, freeAttempt.Object.Fee.IsZero())
	freeOrder := freeAttempt.Object.Order.(*orders.Order)
	require.Equal(t, freeOrder.ID.String(), returnLocation.Query().Get("order_id"))
	appliedFree, err := redemptions.Filter("ID", freeRedemption.ID).Get()
	require.NoError(t, err)
	require.Equal(t, freeOrder.ID, appliedFree.Object.Order.(*orders.Order).ID)
	require.True(t, appliedFree.Object.DiscountAmount.Equal(product.Price))
	require.False(t, appliedFree.Object.Invalidated)
	require.False(t, appliedFree.Object.Counted)

	nextCart := getCart()
	require.NotEqual(t, freeCart.ID, nextCart.ID)
	freeReplay := post(options[0].AccountID.String(), http.StatusSeeOther)
	require.Equal(t, freeCheckout.Header.Get("Location"), freeReplay.Header.Get("Location"))
	count, err = queries.GetQuerySet(&orders.Order{}).Filter("Customer", freeOrder.Customer.ID).Count()
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	require.Len(t, provider.requests, 2)

	// A percentage voucher covers product price. A remaining product fee
	// still needs an account, even with the same 100% coupon.
	_, err = queries.GetQuerySet(&products.Product{}).
		Filter("ID", otherProduct.ID).
		Select("AdditionalFee").
		Update(&products.Product{AdditionalFee: decimal.NewFromInt(1)})
	require.NoError(t, err)
	_, err = queries.GetQuerySet(&carts.CartItem{}).Create(&carts.CartItem{
		Cart:    nextCart,
		Product: otherProduct,
		Amount:  1,
	})
	require.NoError(t, err)
	_, err = redemptions.Create(&coupons.Redemption{
		Cart:   nextCart,
		Coupon: freeCoupon,
	})
	require.NoError(t, err)
	post("", http.StatusBadRequest)
	require.Equal(t, nextCart.ID, getCart().ID)
	require.Len(t, provider.requests, 2)
}
