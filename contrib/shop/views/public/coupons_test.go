//go:build test

package public_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Nigel2392/go-django/contrib/shop/core/shared/api"
	"github.com/Nigel2392/go-django/contrib/shop/core/shared/price"
	"github.com/Nigel2392/go-django/contrib/shop/models/carts"
	"github.com/Nigel2392/go-django/contrib/shop/models/coupons"
	"github.com/Nigel2392/go-django/contrib/shop/models/customers"
	"github.com/Nigel2392/go-django/contrib/shop/models/orders"
	"github.com/Nigel2392/go-django/contrib/shop/models/products"
	"github.com/Nigel2392/go-django/contrib/shop/views/public"
	"github.com/Nigel2392/go-django/djester"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/expr"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-signals"
	"github.com/Nigel2392/mux/v2/middleware/authentication"
	"github.com/Nigel2392/mux/v2/middleware/sessions"
	"github.com/alexedwards/scs/v2"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func testCouponRedemption(d *djester.Tester, t *testing.T, inCart, outsideCart *products.Product) {
	_, _ = d.Logout()
	if _, err := queries.GetQuerySet(&carts.CartItem{}).Delete(); err != nil {
		t.Fatal(err)
	}
	if _, err := queries.GetQuerySet(&carts.Cart{}).Delete(); err != nil {
		t.Fatal(err)
	}
	redeemURL := django.Reverse("api:coupons:redeem")
	seed := func(code string, limit uint64, target *products.Product) *coupons.Coupon {
		t.Helper()
		now := time.Now()
		coupon, err := queries.GetQuerySet(&coupons.Coupon{}).Create(&coupons.Coupon{
			Code: coupons.CouponCode(code), Title: code, UsageLimit: limit,
			MaxDiscount: decimal.NewFromInt(5), ValidFrom: now.Add(-time.Hour), ValidUntil: now.Add(time.Hour),
		})
		if err != nil {
			t.Fatal(err)
		}
		if target != nil {
			attrs.Define(t.Context(), coupon).Get("Products")
			if _, err := coupon.Products.Objects().AddTargets(target); err != nil {
				t.Fatal(err)
			}
		}
		return coupon
	}

	cartWide := seed("CARTWIDE", 0, nil)
	matching := seed("MATCHING", 1, inCart)
	nonmatching := seed("NOMATCH", 0, outsideCart)

	redeem := func(code string, want int, dst any) {
		t.Helper()
		resp, err := d.PostJson(redeemURL, nil, nil, public.RedemptionRequest{CouponCode: code}, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != want {
			var message api.ErrorResponse
			_ = resp.JSON(&message)
			t.Fatalf("redeem %s returned %d, want %d (%s)", code, resp.StatusCode, want, message.Message)
		}
		if dst != nil {
			if err := resp.JSON(dst); err != nil {
				t.Fatal(err)
			}
		}
	}

	resp, err := d.Post(redeemURL, http.Header{"Content-Type": {"application/json"}}, nil, strings.NewReader("{"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("malformed request returned %d", resp.StatusCode)
	}
	redeem("UNKNOWN", http.StatusNotFound, nil)
	redeem(string(cartWide.Code), http.StatusNotFound, nil)

	var added api.DataResponse[public.CartItemResponse]
	resp, err = d.PostJson(django.Reverse("api:carts:add_item"), nil, nil, public.CartItemRequest{
		ProductID: inCart.ID.String(), Amount: 1,
	}, &added)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("adding product returned %d", resp.StatusCode)
	}
	redeem(string(nonmatching.Code), http.StatusNotFound, nil)

	var cart api.DataResponse[struct {
		Object *carts.Cart  `json:"cart"`
		Totals price.Totals `json:"total_price"`
	}]
	redeem(string(cartWide.Code), http.StatusOK, &cart)
	if cart.Data.Object.ID != added.Data.Cart.ID || !cart.Data.Totals.TotalDiscount.Equal(decimal.NewFromInt(5)) {
		t.Fatalf("unexpected discounted cart: %+v", cart)
	}
	redeem(string(cartWide.Code), http.StatusConflict, nil)
	redeem(string(matching.Code), http.StatusOK, &cart)
	if !cart.Data.Totals.TotalDiscount.Equal(decimal.NewFromInt(10)) {
		t.Fatalf("expected both coupons to apply, got discount %s", cart.Data.Totals.TotalDiscount)
	}
	redeem(string(matching.Code), http.StatusNotFound, nil)

	for coupon, want := range map[*coupons.Coupon]int{cartWide: 1, matching: 1, nonmatching: 0} {
		count, err := queries.GetQuerySet(&coupons.Redemption{}).
			Filter("Cart", added.Data.Cart.ID).Filter("Coupon", coupon.ID).Count()
		if err != nil || int(count) != want {
			t.Fatalf("coupon %s has %d redemptions, want %d (err %v)", coupon.Code, count, want, err)
		}
	}
	persisted, err := queries.GetQuerySet(&coupons.Coupon{}).Filter("ID", matching.ID).Get()
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Object.UsageCount != 0 {
		t.Fatalf("cart redemption incremented completed usage to %d", persisted.Object.UsageCount)
	}
}

const (
	couponInvalid = "invalid test coupon"
	couponNoMatch = "no matching test products"
	couponLimit   = "test coupon usage limit reached"
)

type couponTestResponse struct {
	Cart *struct {
		ID     drivers.ULID `json:"id"`
		Active bool         `json:"active"`
	} `json:"cart"`
	Items []struct {
		CartItem struct {
			Amount  uint64 `json:"amount"`
			Product struct {
				ID drivers.ULID `json:"id"`
			} `json:"product"`
		} `json:"item"`
		Redemptions     []couponTestRedemption `json:"coupons"`
		CalculatedPrice price.Totals           `json:"total_price"`
	} `json:"items"`
	Coupons []couponTestRedemption `json:"coupons"`
	Totals  price.Totals           `json:"total_price"`
}

type couponTestRedemption struct {
	ID     uint64
	Coupon struct {
		ID uint64
	}
}

type couponTestClient struct {
	handler http.Handler
	cookie  *http.Cookie
	user    *CartTestUser
}

func newCouponTestClient(t *testing.T) couponTestClient {
	t.Helper()
	store := scs.New()
	ctx, err := store.Load(t.Context(), "")
	if err != nil {
		t.Fatal(err)
	}
	store.Put(ctx, "coupon_test", true)
	token, _, err := store.Commit(ctx)
	if err != nil {
		t.Fatal(err)
	}
	handler := public.RedemptionAPI{
		ErrInvalid:            func(context.Context) string { return couponInvalid },
		ErrNoMatchingProducts: func(context.Context) string { return couponNoMatch },
		ErrRedeemed:           func(context.Context) string { return couponLimit },
	}
	return couponTestClient{
		handler: sessions.SessionMiddleware(store)(http.HandlerFunc(handler.RedeemCode)),
		cookie:  &http.Cookie{Name: store.Cookie.Name, Value: token},
	}
}

func (c couponTestClient) post(t *testing.T, body string, status int, message string) *couponTestResponse {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/carts/coupons/redeem/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(c.cookie)
	ctx := t.Context()
	if c.user != nil {
		ctx = authentication.ContextWithUser(ctx, c.user)
	}
	rec := httptest.NewRecorder()
	c.handler.ServeHTTP(rec, req.WithContext(ctx))
	if rec.Code != status {
		t.Fatalf("redeem returned %d, want %d: %s", rec.Code, status, rec.Body)
	}

	if status != http.StatusOK {
		var got api.ErrorResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decoding coupon error: %v", err)
		}
		if status == http.StatusBadRequest {
			if !strings.HasPrefix(got.Message, message) {
				t.Fatalf("error message %q, want prefix %q", got.Message, message)
			}
		} else if got.Message != message {
			t.Fatalf("error message %q, want %q", got.Message, message)
		}
		return nil
	}

	var got api.DataResponse[couponTestResponse]
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decoding redeemed cart: %v", err)
	}
	return &got.Data
}

func (c couponTestClient) redeem(t *testing.T, coupon *coupons.Coupon, status int, message string) *couponTestResponse {
	t.Helper()
	return c.post(t, fmt.Sprintf(`{"coupon_code":%q}`, coupon.Code), status, message)
}

func (c couponTestClient) cart(t *testing.T, active bool, items ...*products.Product) *carts.Cart {
	t.Helper()
	cart := &carts.Cart{Session: c.cookie.Value, Active: active}
	if c.user != nil {
		cart.User = c.user
	}
	cart, err := queries.GetQuerySet(&carts.Cart{}).Create(cart)
	if err != nil {
		t.Fatal(err)
	}
	for _, product := range items {
		if _, err := queries.GetQuerySet(&carts.CartItem{}).Create(&carts.CartItem{
			Cart: cart, Product: product, Amount: 1,
		}); err != nil {
			t.Fatal(err)
		}
	}
	return cart
}

func newCouponTestCode(t *testing.T, coupon coupons.Coupon, targets ...*products.Product) *coupons.Coupon {
	t.Helper()
	now := time.Now()
	coupon.Code = coupons.CouponCode(drivers.NewULID().String())
	coupon.Title = "Redemption test"
	coupon.ValidFrom = now.Add(-time.Hour)
	coupon.ValidUntil = now.Add(time.Hour)
	created, err := queries.GetQuerySet(&coupons.Coupon{}).Create(&coupon)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 0 {
		attrs.Define(t.Context(), created).Get("Products")
		if _, err := created.Products.Objects().AddTargets(targets...); err != nil {
			t.Fatal(err)
		}
	}
	return created
}

func couponTestRedemptions(t *testing.T, coupon *coupons.Coupon, want int) []*queries.Row[*coupons.Redemption] {
	t.Helper()
	rows, err := queries.GetQuerySet(&coupons.Redemption{}).
		Filter("Coupon", coupon.ID).OrderBy("ID").All()
	if err != nil || len(rows) != want {
		t.Fatalf("coupon %s has %d redemptions, want %d (err %v)", coupon.Code, len(rows), want, err)
	}
	return rows
}

func assertCouponCart(t *testing.T, got *couponTestResponse, cart *carts.Cart, discount string) {
	t.Helper()
	if got.Cart == nil || got.Cart.ID != cart.ID || !got.Cart.Active {
		t.Fatalf("redeemed cart %+v, want active cart %s", got.Cart, cart.ID)
	}
	if !got.Totals.TotalDiscount.Equal(decimal.RequireFromString(discount)) {
		t.Fatalf("discount %s, want %s", got.Totals.TotalDiscount, discount)
	}
}

func testCouponRedemptionCases(t *testing.T, mouse, keyboard, webcam *products.Product) {
	t.Run("request validation", func(t *testing.T) {
		client := newCouponTestClient(t)
		client.cart(t, true, mouse)
		coupon := newCouponTestCode(t, coupons.Coupon{MaxDiscount: decimal.NewFromInt(5)})
		for _, tc := range []struct {
			name, body string
			status     int
		}{
			{"empty body", "", http.StatusBadRequest},
			{"whitespace body", "  ", http.StatusBadRequest},
			{"truncated JSON", `{"coupon_code":`, http.StatusBadRequest},
			{"wrong field", `{"code":"x"}`, http.StatusBadRequest},
			{"unknown field with a valid code", fmt.Sprintf(`{"coupon_code":%q,"extra":true}`, coupon.Code), http.StatusBadRequest},
			{"numeric code", `{"coupon_code":123}`, http.StatusBadRequest},
			{"array code", `{"coupon_code":["x"]}`, http.StatusBadRequest},
			{"array body", `[]`, http.StatusBadRequest},
			{"missing code", `{}`, http.StatusNotFound},
			{"null code", `{"coupon_code":null}`, http.StatusNotFound},
			{"empty code", `{"coupon_code":""}`, http.StatusNotFound},
			{"unrecognized code", `{"coupon_code":"NO-SUCH-COUPON"}`, http.StatusNotFound},
		} {
			t.Run(tc.name, func(t *testing.T) {
				message := couponInvalid
				if tc.status == http.StatusBadRequest {
					message = "failed to read or decode request body:"
				}
				client.post(t, tc.body, tc.status, message)
				couponTestRedemptions(t, coupon, 0)
			})
		}
	})

	t.Run("cart ownership", func(t *testing.T) {
		t.Run("inactive and foreign guest carts are excluded", func(t *testing.T) {
			client, other := newCouponTestClient(t), newCouponTestClient(t)
			client.cart(t, false, mouse)
			foreignCart := other.cart(t, true, mouse)
			coupon := newCouponTestCode(t, coupons.Coupon{MaxDiscount: decimal.NewFromInt(5)})
			client.redeem(t, coupon, http.StatusNotFound, couponInvalid)
			couponTestRedemptions(t, coupon, 0)
			assertCouponCart(t, other.redeem(t, coupon, http.StatusOK, ""), foreignCart, "5")
		})

		t.Run("multiple active carts fail without reserving a use", func(t *testing.T) {
			client := newCouponTestClient(t)
			client.cart(t, true, mouse)
			client.cart(t, true, mouse)
			coupon := newCouponTestCode(t, coupons.Coupon{UsageLimit: 1, MaxDiscount: decimal.NewFromInt(5)})
			client.redeem(t, coupon, http.StatusNotFound, couponInvalid)
			couponTestRedemptions(t, coupon, 0)
		})

		t.Run("authenticated user can redeem on their cart from another session", func(t *testing.T) {
			owner, guest := newCouponTestClient(t), newCouponTestClient(t)
			user, err := queries.GetQuerySet(&CartTestUser{}).Create(&CartTestUser{Name: "coupon-owner", LoggedIn: true})
			if err != nil {
				t.Fatal(err)
			}
			owner.user = user
			cart := owner.cart(t, true, mouse)
			coupon := newCouponTestCode(t, coupons.Coupon{MaxDiscount: decimal.NewFromInt(5)})
			guest.redeem(t, coupon, http.StatusNotFound, couponInvalid)
			couponTestRedemptions(t, coupon, 0)

			guest.user = user
			assertCouponCart(t, guest.redeem(t, coupon, http.StatusOK, ""), cart, "5")
			row := couponTestRedemptions(t, coupon, 1)[0].Object
			if row.Cart.(*carts.Cart).ID != cart.ID || row.Invalidated || row.RedeemedAt.IsZero() || row.Order != nil {
				t.Fatalf("unexpected pending redemption: %+v", row)
			}
		})
	})

	t.Run("usage limits", func(t *testing.T) {
		for _, tc := range []struct {
			name                string
			limit, used         uint64
			active, invalidated bool
			exhausted           bool
		}{
			{name: "completed uses at limit", limit: 2, used: 2, exhausted: true},
			{name: "completed uses exceed limit", limit: 2, used: 3, exhausted: true},
			{name: "completed uses below limit", limit: 2, used: 1},
			{name: "zero limit allows reuse across carts", used: 3, active: true},
			{name: "active reservation on another cart fills limit", limit: 1, active: true, exhausted: true},
			{name: "active reservation below limit", limit: 2, active: true},
			{name: "completed uses and outstanding reservations together fill limit", limit: 2, used: 1, active: true, exhausted: true},
			{name: "invalidated reservation on current cart releases limit", limit: 1, invalidated: true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				client := newCouponTestClient(t)
				cart := client.cart(t, true, mouse)
				coupon := newCouponTestCode(t, coupons.Coupon{
					UsageLimit: tc.limit, UsageCount: tc.used, MaxDiscount: decimal.NewFromInt(5),
				})
				existing := 0
				if tc.active || tc.invalidated {
					reservedCart := cart
					if tc.active {
						reservedCart = newCouponTestClient(t).cart(t, true, mouse)
					}
					if _, err := queries.GetQuerySet(&coupons.Redemption{}).Create(&coupons.Redemption{
						Coupon: coupon, Cart: reservedCart, Invalidated: tc.invalidated, RedeemedAt: time.Now(),
					}); err != nil {
						t.Fatal(err)
					}
					existing++
				}

				if tc.exhausted {
					client.redeem(t, coupon, http.StatusNotFound, couponLimit)
					couponTestRedemptions(t, coupon, existing)
				} else {
					assertCouponCart(t, client.redeem(t, coupon, http.StatusOK, ""), cart, "5")
					rows := couponTestRedemptions(t, coupon, existing+1)
					row := rows[len(rows)-1].Object
					if row.Invalidated || row.Cart.(*carts.Cart).ID != cart.ID {
						t.Fatalf("unexpected new redemption: %+v", row)
					}
					if tc.invalidated && !rows[0].Object.Invalidated {
						t.Fatal("previous redemption was reactivated")
					}
				}

				persisted, err := queries.GetQuerySet(&coupons.Coupon{}).Filter("ID", coupon.ID).Get()
				if err != nil || persisted.Object.UsageCount != tc.used {
					t.Fatalf("unexpected completed usage after redeeming: %+v, err %v", persisted, err)
				}
			})
		}

		t.Run("duplicate preserves the existing reservation", func(t *testing.T) {
			client := newCouponTestClient(t)
			cart := client.cart(t, true, mouse)
			coupon := newCouponTestCode(t, coupons.Coupon{UsageLimit: 3, MaxDiscount: decimal.NewFromInt(5)})
			assertCouponCart(t, client.redeem(t, coupon, http.StatusOK, ""), cart, "5")
			before := couponTestRedemptions(t, coupon, 1)[0].Object
			client.redeem(t, coupon, http.StatusConflict, couponInvalid)
			after := couponTestRedemptions(t, coupon, 1)[0].Object
			if after.ID != before.ID || !after.RedeemedAt.Equal(before.RedeemedAt) {
				t.Fatalf("duplicate modified original redemption: before %+v, after %+v", before, after)
			}
		})
	})

	t.Run("order redemptions", func(t *testing.T) {
		testOrderRedemptionLifecycle(t, mouse)
	})

	t.Run("product scope and applied totals", func(t *testing.T) {
		t.Run("empty cart accepts cart-wide coupon and rejects scoped coupon", func(t *testing.T) {
			client := newCouponTestClient(t)
			cart := client.cart(t, true)
			scoped := newCouponTestCode(t, coupons.Coupon{MaxDiscount: decimal.NewFromInt(5)}, mouse)
			client.redeem(t, scoped, http.StatusNotFound, couponNoMatch)
			couponTestRedemptions(t, scoped, 0)

			wide := newCouponTestCode(t, coupons.Coupon{MaxDiscount: decimal.NewFromInt(5)})
			got := client.redeem(t, wide, http.StatusOK, "")
			assertCouponCart(t, got, cart, "0")
			if len(got.Coupons) != 1 || len(got.Items) != 0 {
				t.Fatalf("unexpected empty cart response: %+v", got)
			}
			couponTestRedemptions(t, wide, 1)
		})

		t.Run("percentage discount applies only to matching lines", func(t *testing.T) {
			client := newCouponTestClient(t)
			cart := client.cart(t, true, mouse, keyboard)
			coupon := newCouponTestCode(t, coupons.Coupon{
				IsPercentage: true, MaxDiscount: decimal.NewFromInt(20),
			}, webcam, keyboard)
			got := client.redeem(t, coupon, http.StatusOK, "")
			assertCouponCart(t, got, cart, "19.90")
			if len(got.Coupons) != 1 || len(got.Items) != 2 {
				t.Fatalf("unexpected coupon response: %+v", got)
			}
			for _, item := range got.Items {
				id := item.CartItem.Product.ID
				switch id {
				case keyboard.ID:
					if len(item.Redemptions) != 1 || item.Redemptions[0].ID != got.Coupons[0].ID ||
						!item.CalculatedPrice.TotalPrice.Equal(decimal.RequireFromString("79.60")) {
						t.Fatalf("matching item did not receive the expected discount: %+v", item)
					}
				case mouse.ID:
					if len(item.Redemptions) != 0 || !item.CalculatedPrice.TotalPrice.Equal(mouse.Price) {
						t.Fatalf("ineligible item received a discount: %+v", item)
					}
				default:
					t.Fatalf("unexpected product %s", id)
				}
			}
		})

		t.Run("oversized fixed discount leaves a zero payable amount", func(t *testing.T) {
			client := newCouponTestClient(t)
			cart := client.cart(t, true, mouse)
			coupon := newCouponTestCode(t, coupons.Coupon{MaxDiscount: decimal.NewFromInt(50)})
			got := client.redeem(t, coupon, http.StatusOK, "")
			assertCouponCart(t, got, cart, "29.99")
			if !got.Totals.Payable().IsZero() || len(got.Items) != 1 ||
				!got.Items[0].CalculatedPrice.LeftOverDiscount.Equal(decimal.RequireFromString("20.01")) {
				t.Fatalf("unexpected totals after oversized discount: %+v", got.Totals)
			}
		})

		t.Run("overlapping coupons with different taxes and quantities", func(t *testing.T) {
			testMixedCouponTotals(t, mouse, webcam)
		})
	})

	t.Run("transaction rollback", func(t *testing.T) {
		for _, during := range []string{"insert", "recalculation"} {
			t.Run(during, func(t *testing.T) {
				client := newCouponTestClient(t)
				cart := client.cart(t, true, mouse)
				coupon := newCouponTestCode(t, coupons.Coupon{UsageLimit: 1, MaxDiscount: decimal.NewFromInt(5)})
				listener, err := queries.SignalPostModelCreate.Listen(t.Context(), func(ctx context.Context, _ signals.Signal[queries.ModelSignal], event queries.ModelSignal) error {
					if _, ok := event.Instance.(*coupons.Redemption); !ok {
						return nil
					}
					if during == "insert" {
						return errors.New("test failure after redemption insert")
					}
					_, err := queries.GetQuerySet(&carts.Cart{}).WithContext(ctx).Filter("ID", cart.ID).
						BulkUpdate(expr.As("Active", expr.Value(false)))
					return err
				})
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					if listener != nil {
						_ = queries.SignalPostModelCreate.Disconnect(context.Background(), listener)
					}
				})

				if during == "insert" {
					client.redeem(t, coupon, http.StatusInternalServerError, "failed to redeem coupon")
				} else {
					client.redeem(t, coupon, http.StatusNotFound, couponInvalid)
				}
				if err := queries.SignalPostModelCreate.Disconnect(t.Context(), listener); err != nil {
					t.Fatal(err)
				}
				listener = nil
				couponTestRedemptions(t, coupon, 0)
				assertCouponCart(t, client.redeem(t, coupon, http.StatusOK, ""), cart, "5")
				couponTestRedemptions(t, coupon, 1)
			})
		}
	})
}

func testOrderRedemptionLifecycle(t *testing.T, product *products.Product) {
	t.Helper()
	ctx := t.Context()
	customer, err := queries.GetQuerySet(&customers.Customer{}).Create(&customers.Customer{
		Email:     drivers.MustParseEmail("redemptions@example.test"),
		FirstName: "Coupon",
		LastName:  "Customer",
	})
	require.NoError(t, err)

	ordersQuery := queries.GetQuerySet(&orders.Order{})
	redemptionsQuery := queries.GetQuerySet(&coupons.Redemption{})
	prepare := func(t *testing.T, coupon *coupons.Coupon) (*orders.Order, *coupons.Redemption) {
		t.Helper()
		order, err := ordersQuery.Create(&orders.Order{
			Customer: customer,
			State:    "order_pending",
		})
		require.NoError(t, err)

		reservation, err := redemptionsQuery.Create(&coupons.Redemption{
			Coupon:   coupon,
			Order:    order,
			Customer: customer,
		})
		require.NoError(t, err)
		return order, reservation
	}

	t.Run("completion moves a reservation into the usage count once", func(t *testing.T) {
		coupon := newCouponTestCode(t, coupons.Coupon{
			UsageLimit:  2,
			MaxDiscount: decimal.NewFromInt(5),
		})
		order, reservation := prepare(t, coupon)
		require.NoError(t, orders.ReconcileCouponRedemptions(ctx, order.ID))
		row := couponTestRedemptions(t, coupon, 1)[0].Object
		require.False(t, row.Counted)
		require.False(t, row.Invalidated)

		client := newCouponTestClient(t)
		cart := client.cart(t, true, product)
		assertCouponCart(t, client.redeem(t, coupon, http.StatusOK, ""), cart, "5")

		txCtx, tx, err := queries.StartTransaction(ctx)
		require.NoError(t, err)
		defer tx.Rollback(txCtx)
		_, err = ordersQuery.WithContext(txCtx).Filter("ID", order.ID).Select("Completed").Update(&orders.Order{
			Completed: true,
		})
		require.NoError(t, err)
		require.NoError(t, orders.ReconcileCouponRedemptions(txCtx, order.ID))
		require.NoError(t, tx.Rollback(txCtx))
		persisted, err := queries.GetQuerySet(&coupons.Coupon{}).Filter("ID", coupon.ID).Get()
		require.NoError(t, err)
		require.Zero(t, persisted.Object.UsageCount)
		row = couponTestRedemptions(t, coupon, 2)[0].Object
		require.False(t, row.Counted)

		_, err = ordersQuery.Filter("ID", order.ID).Select("Completed").Update(&orders.Order{
			Completed: true,
		})
		require.NoError(t, err)
		require.NoError(t, orders.ReconcileCouponRedemptions(ctx, order.ID))
		require.NoError(t, orders.ReconcileCouponRedemptions(ctx, order.ID))

		row = couponTestRedemptions(t, coupon, 2)[0].Object
		require.Equal(t, reservation.ID, row.ID)
		require.True(t, row.Counted)
		require.False(t, row.Invalidated)
		require.True(t, row.DiscountAmount.IsZero(), "historical discount amount is unknown")
		persisted, err = queries.GetQuerySet(&coupons.Coupon{}).Filter("ID", coupon.ID).Get()
		require.NoError(t, err)
		require.EqualValues(t, 1, persisted.Object.UsageCount)

		_, err = redemptionsQuery.Filter("Cart", cart.ID).Select("Invalidated").Update(&coupons.Redemption{
			Invalidated: true,
		})
		require.NoError(t, err)
		client = newCouponTestClient(t)
		cart = client.cart(t, true, product)
		assertCouponCart(t, client.redeem(t, coupon, http.StatusOK, ""), cart, "5")

		client = newCouponTestClient(t)
		client.cart(t, true, product)
		client.redeem(t, coupon, http.StatusNotFound, couponLimit)
	})

	t.Run("rejection releases only outstanding reservations", func(t *testing.T) {
		coupon := newCouponTestCode(t, coupons.Coupon{
			UsageLimit:  2,
			MaxDiscount: decimal.NewFromInt(5),
		})
		order, reservation := prepare(t, coupon)
		other, used := prepare(t, coupon)
		_, err := ordersQuery.Filter("ID", other.ID).Select("Completed").Update(&orders.Order{
			Completed: true,
		})
		require.NoError(t, err)
		require.NoError(t, orders.ReconcileCouponRedemptions(ctx, other.ID))
		_, err = ordersQuery.Filter("ID__in", []drivers.ULID{order.ID, other.ID}).Select("State").Update(&orders.Order{
			State: "order_rejected",
		})
		require.NoError(t, err)
		require.NoError(t, orders.ReconcileCouponRedemptions(ctx, order.ID))
		require.NoError(t, orders.ReconcileCouponRedemptions(ctx, other.ID))

		rows := couponTestRedemptions(t, coupon, 2)
		require.Equal(t, reservation.ID, rows[0].Object.ID)
		require.True(t, rows[0].Object.Invalidated)
		require.False(t, rows[0].Object.Counted)
		require.Equal(t, order.ID, rows[0].Object.Order.(*orders.Order).ID)
		require.Equal(t, used.ID, rows[1].Object.ID)
		require.False(t, rows[1].Object.Invalidated)
		require.True(t, rows[1].Object.Counted)

		client := newCouponTestClient(t)
		cart := client.cart(t, true, product)
		assertCouponCart(t, client.redeem(t, coupon, http.StatusOK, ""), cart, "5")
	})
}

func testMixedCouponTotals(t *testing.T, seed, unrelated *products.Product) {
	t.Helper()
	paper := createTestProduct(t, seed.Category, "Coupon paper", "coupon-paper", "10", "0", "0")
	pens := createTestProduct(t, seed.Category, "Coupon pens", "coupon-pens", "20", "21", "0")
	ink := createTestProduct(t, seed.Category, "Coupon ink", "coupon-ink", "50", "9", "0")

	client := newCouponTestClient(t)
	cart := client.cart(t, true)
	for _, line := range []struct {
		product *products.Product
		amount  uint64
	}{{paper, 4}, {pens, 2}, {ink, 3}} {
		if _, err := queries.GetQuerySet(&carts.CartItem{}).Create(&carts.CartItem{
			Cart: cart, Product: line.product, Amount: line.amount,
		}); err != nil {
			t.Fatal(err)
		}
	}

	wideFixed := newCouponTestCode(t, coupons.Coupon{MaxDiscount: decimal.NewFromInt(5)})
	widePercent := newCouponTestCode(t, coupons.Coupon{IsPercentage: true, MaxDiscount: decimal.NewFromInt(10)})
	pensFixed := newCouponTestCode(t, coupons.Coupon{MaxDiscount: decimal.NewFromInt(4)}, pens)
	inkFixed := newCouponTestCode(t, coupons.Coupon{MaxDiscount: decimal.NewFromInt(10)}, ink)
	sharedPercent := newCouponTestCode(t, coupons.Coupon{IsPercentage: true, MaxDiscount: decimal.NewFromInt(20)}, pens, ink)
	unmatched := newCouponTestCode(t, coupons.Coupon{MaxDiscount: decimal.NewFromInt(15)}, unrelated)

	var got *couponTestResponse
	for _, coupon := range []*coupons.Coupon{wideFixed, widePercent, pensFixed, inkFixed, sharedPercent} {
		got = client.redeem(t, coupon, http.StatusOK, "")
		couponTestRedemptions(t, coupon, 1)
	}
	client.redeem(t, unmatched, http.StatusNotFound, couponNoMatch)
	couponTestRedemptions(t, unmatched, 0)

	assertCouponCart(t, got, cart, "75.30")
	if len(got.Coupons) != 5 || len(got.Items) != 3 ||
		!got.Totals.TotalPrice.Equal(decimal.RequireFromString("154.70")) ||
		!got.Totals.TotalTax.Equal(decimal.RequireFromString("14.112")) ||
		!got.Totals.Payable().Equal(decimal.RequireFromString("168.812")) {
		t.Fatalf("unexpected mixed cart totals: %+v", got)
	}
	var cartCouponIDs []uint64
	for _, redemption := range got.Coupons {
		cartCouponIDs = append(cartCouponIDs, redemption.Coupon.ID)
	}
	slices.Sort(cartCouponIDs)
	if !slices.Equal(cartCouponIDs, []uint64{wideFixed.ID, widePercent.ID, pensFixed.ID, inkFixed.ID, sharedPercent.ID}) {
		t.Fatalf("unexpected mixed cart coupons: %v", cartCouponIDs)
	}

	type wantLine struct {
		amount               uint64
		price, tax, discount string
		couponIDs            []uint64
	}
	wants := map[drivers.ULID]wantLine{
		paper.ID: {4, "31.50", "0", "8.50", nil},
		pens.ID:  {2, "25.20", "5.292", "14.80", []uint64{pensFixed.ID, sharedPercent.ID}},
		ink.ID:   {3, "98", "8.82", "52", []uint64{inkFixed.ID, sharedPercent.ID}},
	}
	for _, item := range got.Items {
		id := item.CartItem.Product.ID
		want, ok := wants[id]
		if !ok {
			t.Fatalf("unexpected or repeated line %s", id)
		}
		delete(wants, id)
		var applied []uint64
		for _, redemption := range item.Redemptions {
			applied = append(applied, redemption.Coupon.ID)
		}
		slices.Sort(applied)
		line := item.CalculatedPrice
		if item.CartItem.Amount != want.amount || !slices.Equal(applied, want.couponIDs) ||
			!line.TotalPrice.Equal(decimal.RequireFromString(want.price)) ||
			!line.TotalTax.Equal(decimal.RequireFromString(want.tax)) ||
			!line.TotalDiscount.Equal(decimal.RequireFromString(want.discount)) {
			t.Fatalf("product %s: amount %d, coupons %v, totals %+v, want %+v",
				id, item.CartItem.Amount, applied, line, want)
		}
	}
}
