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
	"github.com/Nigel2392/go-django/contrib/shop/shared/api"
	"github.com/Nigel2392/go-django/contrib/shop/shared/price"
	"github.com/Nigel2392/go-django/contrib/shop/util/paystate"
	"github.com/Nigel2392/go-django/contrib/shop/views/public"
	"github.com/Nigel2392/go-django/djester"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/currency"
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
	hostedPSP := &payments.PSP{
		Name:           "checkout-endpoint",
		TextLabel:      func(context.Context) string { return "Hosted payment" },
		CurrencyUnit:   currency.EUR,
		FixedFeeAmount: decimal.RequireFromString("0.50"),
		Factory: func(context.Context, *payments.ProviderAccount) (payments.Provider, error) {
			return provider, nil
		},
	}
	payments.RegisterProvider(hostedPSP)
	payments.RegisterProvider(&payments.PSP{
		Name:         "checkout-endpoint-other",
		TextLabel:    func(context.Context) string { return "Other payment" },
		CurrencyUnit: currency.EUR,
		Factory: func(context.Context, *payments.ProviderAccount) (payments.Provider, error) {
			return provider, nil
		},
	})
	payments.RegisterProvider(&payments.PSP{
		Name:         "checkout-endpoint-disabled",
		TextLabel:    func(context.Context) string { return "Disabled payment" },
		CurrencyUnit: currency.EUR,
		Factory: func(context.Context, *payments.ProviderAccount) (payments.Provider, error) {
			return provider, nil
		},
	})
	accounts := queries.GetQuerySet(&payments.ProviderAccount{})
	account, err := accounts.Create(&payments.ProviderAccount{
		InternalName: "checkout-endpoint",
		Enabled:      true,
	})
	require.NoError(t, err)
	other, err := accounts.Create(&payments.ProviderAccount{
		InternalName: "checkout-endpoint-other",
		Enabled:      true,
	})
	require.NoError(t, err)
	disabled, err := accounts.ExplicitSave().Create(&payments.ProviderAccount{
		InternalName: "checkout-endpoint-disabled",
		Enabled:      false,
	})
	require.NoError(t, err)

	checkoutPath := django.Reverse("api:payments:checkout")
	optionsPath := django.Reverse("api:payments:options")
	cartPath := django.Reverse("api:carts")
	addItemPath := django.Reverse("api:carts:add_item")
	customerPath := django.Reverse("api:carts:customer")

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

	getCart := func() *carts.Cart {
		t.Helper()
		var body api.DataResponse[struct {
			Cart *carts.Cart `json:"cart"`
		}]
		response, err := d.Get(cartPath, nil, nil)
		require.NoError(t, err)
		defer response.Body.Close()
		require.Equal(t, http.StatusOK, response.StatusCode)
		require.NoError(t, response.JSON(&body))
		require.NotNil(t, body.Data.Cart)
		return body.Data.Cart
	}

	// Clean carts before testing guest checkout
	_, _ = d.Logout()
	_, _ = queries.GetQuerySet(&carts.CartItem{}).Delete()
	_, _ = queries.GetQuerySet(&carts.Cart{}).Delete()

	// 1. Guest without any cart -> 404 Not Found
	post(account.ID.String(), http.StatusNotFound)
	post("invalid", http.StatusBadRequest)

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

	// 2. Guest 4-step checkout flow without logging in!
	// Step 1: Add item to guest cart
	var guestAddResp api.DataResponse[public.CartItemResponse]
	addResp, err := d.PostJson(addItemPath, nil, nil, public.CartItemRequest{
		ProductID: product.ID.String(),
		Amount:    1,
	}, &guestAddResp)
	require.NoError(t, err)
	addResp.Body.Close()
	require.Equal(t, http.StatusOK, addResp.StatusCode)
	guestCartID := guestAddResp.Data.Cart.ID

	// Attempting checkout as guest before setting customer details -> 422 Unprocessable Entity
	post(account.ID.String(), http.StatusUnprocessableEntity)
	require.Empty(t, provider.requests)

	// Step 2: Set customer details on guest cart via POST /api/carts/customer
	var guestCartResp api.DataResponse[public.CartResponse]
	custResp, err := d.PostJson(customerPath, nil, nil, public.CartCustomerRequest{
		Email:     "guest-checkout@example.test",
		FirstName: "Guest",
		LastName:  "Buyer",
	}, &guestCartResp)
	require.NoError(t, err)
	custResp.Body.Close()
	require.Equal(t, http.StatusOK, custResp.StatusCode)
	require.NotNil(t, guestCartResp.Data.Cart.Customer)
	guestCustomerID := guestCartResp.Data.Cart.Customer.ID

	// Step 3: Fetch PSP list & quotes (GET /api/payments/options)
	optResp, err := d.Get(optionsPath, nil, nil)
	require.NoError(t, err)
	var guestOpts api.DataResponse[[]public.PaymentOption]
	require.NoError(t, optResp.JSON(&guestOpts))
	optResp.Body.Close()
	require.Equal(t, http.StatusOK, optResp.StatusCode)
	require.NotEmpty(t, guestOpts.Data)

	// Step 4: Submit PSP (POST /api/payments/checkout) as guest
	guestCheckoutResp := post(account.ID.String(), http.StatusSeeOther)
	require.Equal(t, "https://checkout.example.test/pay", guestCheckoutResp.Header.Get("Location"))
	require.Len(t, provider.requests, 1)
	guestPayReq := provider.requests[0]
	provider.requests = nil

	guestSavedCart, err := queries.GetQuerySet(&carts.Cart{}).Filter("ID", guestCartID).Get()
	require.NoError(t, err)
	require.False(t, guestSavedCart.Object.Active)

	guestPayment, err := queries.GetQuerySet(&payments.Payment{}).Filter("ID", guestPayReq.AttemptID).Select("*", "Order.*").Get()
	require.NoError(t, err)
	guestOrder := guestPayment.Object.Order.(*orders.Order)
	require.Equal(t, guestCustomerID, guestOrder.Customer.ID)

	// 3. Authenticated checkout flow with coupon redemptions and PSP retry before cart conversion
	login, err := d.Login("checkout_user")
	require.NoError(t, err)
	login.Body.Close()

	post(account.ID.String(), http.StatusNotFound)

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

	post(drivers.NewULID().String(), http.StatusNotFound)
	post(disabled.ID.String(), http.StatusNotFound)
	require.Empty(t, provider.requests)

	ordersBefore, err := queries.GetQuerySet(&orders.Order{}).Count()
	require.NoError(t, err)
	paymentsBefore, err := queries.GetQuerySet(&payments.Payment{}).Count()
	require.NoError(t, err)

	// When provider.startErr is set -> 502 Bad Gateway, cart stays Active: true, NO Order or Payment row created!
	provider.startErr = errors.New("provider did not respond")
	post(account.ID.String(), http.StatusBadGateway)
	require.Len(t, provider.requests, 1)
	require.True(t, provider.requests[0].Amount.Amount.Equal(decimal.RequireFromString("30.74")))
	provider.requests = nil

	savedCart, err := queries.GetQuerySet(&carts.Cart{}).Filter("ID", cart.ID).Get()
	require.NoError(t, err)
	require.True(t, savedCart.Object.Active, "cart must stay active when PSP CreatePayment fails")

	ordersAfterFail, err := queries.GetQuerySet(&orders.Order{}).Count()
	require.NoError(t, err)
	require.Equal(t, ordersBefore, ordersAfterFail, "no Order row should be created when PSP fails")
	paymentsAfterFail, err := queries.GetQuerySet(&payments.Payment{}).Count()
	require.NoError(t, err)
	require.Equal(t, paymentsBefore, paymentsAfterFail, "no Payment row should be created when PSP fails")

	// User can still fetch payment options on the same active cart
	optResp2, err := d.Get(optionsPath, nil, nil)
	require.NoError(t, err)
	optResp2.Body.Close()
	require.Equal(t, http.StatusOK, optResp2.StatusCode)

	// Now provider recovers -> checkout succeeds (303 See Other), converts cart, creates Order + Payment, attaches coupons
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
	require.Len(t, provider.requests, 1)
	create := provider.requests[0]
	require.False(t, create.AttemptID.IsZero())
	require.True(t, create.Amount.Amount.Equal(decimal.RequireFromString("30.74")), "got %s", create.Amount.Amount)
	require.Equal(t, "https://merchant.example.test/order/thanks?payment_id="+create.AttemptID.String(), create.ReturnURL)
	require.Equal(t, "https://merchant.example.test"+django.Reverse("api:payments:webhook", account.ID), create.WebhookURL)

	savedCart, err = queries.GetQuerySet(&carts.Cart{}).Filter("ID", cart.ID).Get()
	require.NoError(t, err)
	require.False(t, savedCart.Object.Active)

	attempt, err := queries.GetQuerySet(&payments.Payment{}).Filter("ID", create.AttemptID).Select("*", "Order.*").Get()
	require.NoError(t, err)
	require.True(t, attempt.Object.TransactionID.Valid)
	require.Equal(t, "test-reference", attempt.Object.TransactionID.String)
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

	count, err := queries.GetQuerySet(&orders.Order{}).Filter("Customer", customer.Object.ID).Count()
	require.NoError(t, err)
	require.EqualValues(t, 1, count)

	provider.status.State = paystate.STATUS_PAID
	provider.status.ProviderFee = &price.Money{
		Amount:   decimal.RequireFromString("0.28"),
		Currency: currency.USD,
	}
	_, err = payments.Refresh(ctx, create.AttemptID)
	require.NoError(t, err)
	savedPayment, err := payments.Refresh(ctx, create.AttemptID)
	require.NoError(t, err)
	require.Len(t, savedPayment.History.Data, 2)
	require.Equal(t, payments.EventCheckoutOpened, savedPayment.History.Data[0].Kind)
	require.Equal(t, payments.EventStatusUpdated, savedPayment.History.Data[1].Kind)
	require.Equal(t, paystate.STATUS_PAID, savedPayment.History.Data[1].State)
	require.True(t, savedPayment.Fee.Equal(decimal.RequireFromString("0.50")))
	require.True(t, savedPayment.Amount.Equal(create.Amount.Amount))
	require.NotNil(t, savedPayment.ProviderFee.Data)
	require.Equal(t, currency.USD, savedPayment.ProviderFee.Data.Currency)
	require.True(t, savedPayment.ProviderFee.Data.Amount.Equal(decimal.RequireFromString("0.28")))

	// 4. Zero-total free checkout (100% coupon)
	_, _ = d.Logout()
	login, err = d.Login("other_checkout_user")
	require.NoError(t, err)
	login.Body.Close()
	post(account.ID.String(), http.StatusNotFound)
	require.Len(t, provider.requests, 1)

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
	optionsResponse, err := d.Get(optionsPath, nil, nil)
	require.NoError(t, err)
	defer optionsResponse.Body.Close()
	require.Equal(t, http.StatusOK, optionsResponse.StatusCode)
	var optionsResp api.DataResponse[[]public.PaymentOption]
	require.NoError(t, optionsResponse.JSON(&optionsResp))
	options := optionsResp.Data
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
	freePaymentIDStr := returnLocation.Query().Get("payment_id")
	require.NotEmpty(t, freePaymentIDStr)
	var freePaymentID drivers.ULID
	require.NoError(t, freePaymentID.UnmarshalText([]byte(freePaymentIDStr)))

	freeAttempt, err := queries.GetQuerySet(&payments.Payment{}).
		Filter("ID", freePaymentID).
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
	count, err = queries.GetQuerySet(&orders.Order{}).Filter("Customer", freeOrder.Customer.ID).Count()
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	require.Len(t, provider.requests, 1)

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
	require.Len(t, provider.requests, 1)
}
