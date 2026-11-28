package public

import (
	"context"
	"math/rand"
	"net/http"
	"time"

	"github.com/Nigel2392/go-django/contrib/shop/core/shared/api"
	"github.com/Nigel2392/go-django/contrib/shop/models/carts"
	"github.com/Nigel2392/go-django/contrib/shop/models/coupons"
	"github.com/Nigel2392/go-django/contrib/shop/models/products"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/logger"
)

type RedemptionRequest struct {
	CouponCode string `json:"coupon_code"`
}

type RedemptionAPI struct {
	ErrInvalid            func(ctx context.Context) string
	ErrNoMatchingProducts func(ctx context.Context) string
	ErrRedeemed           func(ctx context.Context) string
}

var (
	minLagMS = 5
	maxLagMS = 100
)

// induce lag into the API so enumerating hits on the coupon code is harder
func laggy() {
	time.Sleep(
		time.Millisecond * time.Duration(rand.Intn(maxLagMS-minLagMS)+minLagMS),
	)
}

func (a RedemptionAPI) RedeemCode(w http.ResponseWriter, r *http.Request) {
	var (
		req = RedemptionRequest{}
		err = api.Decode(r, &req)
	)
	if err != nil {
		api.RespondError(w,
			http.StatusBadRequest,
			"failed to read or decode request body: %v", err,
		)
		return
	}

	couponQuery := queries.
		GetQuerySet(&coupons.Coupon{}).
		Filter(
			expr.Q("Code", req.CouponCode),
		)
	// Identify the rows before starting the transaction. The first writes in
	// the transaction can then lock those rows before any snapshot reads.
	candidate, err := couponQuery.
		WithContext(r.Context()).
		Select("ID").
		Get()
	if err != nil {
		laggy()
		if !errors.Is(err, errors.NoRows) {
			logger.Warnf("error fetching coupon with code %q: %v", req.CouponCode, err)
		}
		api.RespondError(w, http.StatusNotFound, a.ErrInvalid(r.Context()))
		return
	}

	activeCart, err := cartQuery(r.Context(), r, false).
		Select("ID").
		Get()
	if err != nil {
		if !errors.Is(err, errors.NoRows) {
			logger.Warnf("error fetching cart for coupon request: %v", err)
		}
		api.RespondError(w, http.StatusNotFound, a.ErrInvalid(r.Context()))
		return
	}

	ctx, tx, err := queries.StartTransaction(r.Context())
	if err != nil {
		api.RespondError(w,
			http.StatusInternalServerError,
			"failed to start transaction: %v", err,
		)
		return
	}

	defer tx.Rollback(ctx)

	// Acquire the same cart lock as checkout before reserving a coupon.
	_, err = carts.
		Query().
		WithContext(ctx).
		Filter(
			expr.Q("ID", activeCart.Object.ID),
			expr.Q("Active", true),
		).
		BulkUpdate(expr.As("Active", expr.Value(true)))
	if err != nil {
		logger.Warnf("error locking cart for coupon request: %v", err)
		api.RespondError(w, http.StatusInternalServerError, "failed to redeem coupon")
		return
	}

	couponQuery = couponQuery.
		WithContext(ctx).
		Filter(
			expr.Q("ID", candidate.Object.ID),
		)

	// Serialize reservations and completion on the coupon row so checking
	// capacity and reserving it are one transaction.
	_, err = couponQuery.
		BulkUpdate(expr.As("UsageCount", expr.Field("UsageCount")))
	if err != nil {
		logger.Warnf("error locking coupon for redemption: %v", err)
		api.RespondError(w, http.StatusInternalServerError, "failed to redeem coupon")
		return
	}

	couponRow, err := couponQuery.
		Preload(queries.NoJoins("Products")).
		Get()
	if err != nil {
		// induce lag into the API so enumerating hits on the coupon code is harder
		laggy()

		if !errors.Is(err, errors.NoRows) {
			logger.Warnf("error fetching coupon with code %q: %v", req.CouponCode, err)
		}

		api.RespondError(w,
			http.StatusNotFound,
			a.ErrInvalid(ctx),
		)
		return
	}

	// Completion moves a reservation into UsageCount.
	var coupon = couponRow.Object
	if coupon.UsageLimit > 0 && coupon.UsageCount >= coupon.UsageLimit {
		api.RespondError(w,
			http.StatusNotFound,
			a.ErrRedeemed(ctx),
		)
		return
	}

	// filter all active redemptions
	var redemptionsQs = queries.
		GetQuerySet(&coupons.Redemption{}).
		WithContext(ctx).
		Filter(
			expr.Q("Invalidated", false),
			expr.Q("Coupon", coupon),
		)

	if coupon.UsageLimit > 0 {
		totalRedemptions, err := redemptionsQs.
			Filter(
				expr.Q("Counted", false),
			).
			Count()
		if err != nil {
			// induce lag into the API so enumerating hits on the coupon code is harder
			laggy()

			if !errors.Is(err, errors.NoRows) {
				logger.Warnf("error fetching cart with coupon request code %q: %v", req.CouponCode, err)
			}

			api.RespondError(w,
				http.StatusNotFound,
				a.ErrInvalid(ctx),
			)
			return
		}

		if uint64(totalRedemptions) >= coupon.UsageLimit-coupon.UsageCount {
			api.RespondError(w,
				http.StatusNotFound,
				a.ErrRedeemed(ctx),
			)
			return
		}
	}

	cartQ := cartQuery(ctx, r, true).
		Filter(
			expr.Q("ID", activeCart.Object.ID),
		)
	cartRow, err := cartQ.Get()
	if err != nil {
		if !errors.Is(err, errors.NoRows) {
			logger.Warnf("error fetching cart for coupon request: %v", err)
		}
		api.RespondError(w, http.StatusNotFound, a.ErrInvalid(ctx))
		return
	}

	// check if redemption exists for cart or not, i.e.
	// if the user has already redeemed this code
	exists, err := redemptionsQs.
		Filter("Cart", cartRow.Object.ID).
		Exists()

	switch {
	case !exists && err == nil:
		// happy case, user has not already redeemed the code

	case err != nil:
		// unhappy case, error other than [errors.NoRows] during redemption query
		laggy()

		logger.Warnf("error fetching coupon redemption with code %q: %v", req.CouponCode, err)

		api.RespondError(w,
			http.StatusNotFound,
			a.ErrInvalid(ctx),
		)
		return

	case exists:
		// unhappy case, coupon is already redeemed for the cart
		api.RespondError(w,
			http.StatusConflict,
			a.ErrInvalid(ctx),
		)
		return
	}

	redemption := &coupons.Redemption{
		Coupon:     coupon,
		Cart:       cartRow.Object,
		RedeemedAt: time.Now(),
	}

	// validate if there are products bound to the coupon
	// if there are, check if the any of the
	// cart items matches any of the products
	couponProducts := coupon.Products.AsList()
	if len(couponProducts) > 0 {
		productsFromCoupon := make(map[string]*products.Product)
		for _, product := range couponProducts {
			productsFromCoupon[product.Object.ID.String()] = product.Object
		}

		// check if the cart has any products that matches a bound coupon product
		for _, cartItem := range cartRow.Object.CartItems.AsList() {
			if cartItem.Product == nil {
				continue
			}
			_, ok := productsFromCoupon[cartItem.Product.ID.String()]
			if ok {
				goto createRedemption
			}
		}

		// respond an error indicating that
		// no product in the cart matched
		api.RespondError(w,
			http.StatusNotFound,
			a.ErrNoMatchingProducts(ctx),
		)
		return
	}

createRedemption:
	_, err = queries.
		GetQuerySet(&coupons.Redemption{}).
		WithContext(ctx).
		Create(redemption)
	if err != nil {
		logger.Warnf("error creating coupon redemption: %v", err)
		api.RespondError(w, http.StatusInternalServerError, "failed to redeem coupon")
		return
	}

	cartsRow, err := cartQ.CalcOne()
	if err != nil {
		logger.Warnf("error calculating cart with applied coupon: %v", err)
		api.RespondError(w,
			http.StatusNotFound,
			a.ErrInvalid(ctx),
		)
		return
	}

	err = coupons.Signals.Redeemed.Send(ctx, coupons.CouponRedeemedSignal{
		Request: r,
		Coupon:  coupon,
		Cart:    cartRow.Object,
	})
	if err != nil {
		logger.Warnf("error sending coupon redemption signal: %v", err)
		api.RespondError(w, http.StatusInternalServerError, "failed to redeem coupon")
		return
	}

	if err := tx.Commit(ctx); err != nil {
		logger.Warnf("error committing coupon redemption: %v", err)
		api.RespondError(w, http.StatusInternalServerError, "failed to redeem coupon")
		return
	}

	// encode the response
	err = api.Respond(w, cartsRow)
	if err != nil {
		except.Fail(
			http.StatusInternalServerError,
			"failed to encode the cart item response: %v", err,
		)
	}
}
