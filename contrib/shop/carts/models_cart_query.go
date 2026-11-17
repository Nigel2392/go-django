package carts

import (
	"iter"
	"net/http"
	"sort"
	"strings"

	"github.com/Nigel2392/go-django/contrib/shop/coupons"
	"github.com/Nigel2392/go-django/contrib/shop/products"
	"github.com/Nigel2392/go-django/contrib/shop/shared/price"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/mux/middleware/authentication"
	"github.com/Nigel2392/mux/middleware/sessions"
	"github.com/shopspring/decimal"
)

type CartQuery struct {
	*queries.WrappedQuerySet[*Cart, *CartQuery, *queries.QuerySet[*Cart]]
}

type CartItemRow struct {
	CartItem        *CartItem
	Redemptions     []*coupons.Redemption
	CalculatedPrice price.Totals
}

type CartRow struct {
	*queries.Row[*Cart]
	Items   []*CartItemRow
	Coupons []*coupons.Redemption
	Totals  price.Totals
}

func Query() *CartQuery {
	wrappedQuerySet := &CartQuery{}
	wrappedQuerySet.WrappedQuerySet = queries.WrapQuerySet[*Cart](
		queries.GetQuerySet(&Cart{}),
		wrappedQuerySet,
	)
	return wrappedQuerySet
}

func (qs *CartQuery) CloneQuerySet(wrapped *queries.WrappedQuerySet[*Cart, *CartQuery, *queries.QuerySet[*Cart]]) *CartQuery {
	return &CartQuery{
		WrappedQuerySet: wrapped,
	}
}

func (qs *CartQuery) CartForRequest(r *http.Request) *CartQuery {
	var user = authentication.Retrieve(r)
	var session = sessions.Retrieve(r)
	var baseQ expr.ClauseExpression = expr.Q("Session", session.Token())
	if user.IsAuthenticated() {
		baseQ = baseQ.Or(expr.Q("User", user))
	}

	return qs.
		Clone().
		Filter(baseQ.And(expr.Q("Active", true))).
		OrderBy("CreatedAt")
}

func (qs *CartQuery) PreloadProducts(productFields ...string) *CartQuery {
	var flds = []any{"*"}
	if len(productFields) == 0 {
		flds = append(flds, "Product.*")
	} else {
		for _, fld := range productFields {
			flds = append(
				flds, strings.Join([]string{"Product", fld}, "."),
			)
		}
	}

	return qs.Clone().Preload(queries.Preload{
		Path:    "CartItems",
		NoJoins: true,
		QuerySet: queries.
			GetQuerySet[attrs.Definer](&CartItem{}).
			Select(
				append([]any{"*"}, flds...)...,
			),
	})
}

type cartRedemption struct {
	l []*coupons.Redemption
}

func (qs *CartQuery) Calc() (int, iter.Seq2[*CartRow, error], error) {
	qs = qs.Clone()

	tx, err := qs.StartTransaction(qs.Context())
	if err != nil {
		return 0, nil, errors.Wrap(err, "CartQuery.Calc[StartTransaction]")
	}

	defer tx.Rollback(qs.Context())

	redemptionsQs := queries.
		GetQuerySet(&coupons.Redemption{}).
		WithContext(qs.Context()).
		Filter("Cart__in", qs.Select("ID")).
		Filter("Coupon.Products__in", qs.Select("CartItems.Product")).
		Preload("Coupon.Products")

	_, ri, err := redemptionsQs.IterAll()
	if err != nil {
		return 0, nil, errors.Wrap(err, "CartQuery.Calc[coupons.Redemptions/IterAll]")
	}

	cartRedemptions := make(map[drivers.ULID][]*coupons.Redemption)
	for row, err := range ri {
		if err != nil {
			return 0, nil, errors.Wrap(err, "CartQuery.Calc[coupons.Redemptions/IterAll/Rows]")
		}

		if row.Object.Cart == nil {
			logger.Warnf("cart not found for redemption ID %v", row.Object.ID)
			continue
		}

		cartObj, ok := row.Object.Cart.(*Cart)
		if !ok {
			logger.Warnf("object %T is not of type *Cart", row.Object.Cart)
			continue
		}

		cr, ok := cartRedemptions[cartObj.ID]
		if !ok {
			cr = make([]*coupons.Redemption, 0, 1)
		}

		cartRedemptions[cartObj.ID] = append(cr, row.Object)
	}

	qs = qs.Preload(queries.Preload{
		Path:    "CartItems",
		NoJoins: true,
		QuerySet: queries.
			GetQuerySet[attrs.Definer](&CartItem{}).
			Select(
				"*",
				"Product.ID",
				"Product.Price",
				"Product.ItemFee",
				"Product.TaxPercentage",
			).
			OrderBy(
				"Product.Price",
			), // Order by price ascending (cheapest first) or descending based on preference
	})

	n, cartRows, err := qs.IterAll()
	if err != nil {
		return 0, nil, errors.Wrap(err, "CartQuery.Calc[IterAll]")
	}

	seq := func(yield func(*CartRow, error) bool) {
		for row, err := range cartRows {
			if err != nil {
				yield(nil, errors.Wrap(err, "CartQuery.Calc[IterAll/Rows]"))
				return
			}

			items := row.Object.CartItems.AsList()
			cartRow := &CartRow{
				Row:     row,
				Items:   make([]*CartItemRow, 0, len(items)),
				Coupons: make([]*coupons.Redemption, 0),
			}

			var (
				redemptions               = cartRedemptions[row.Object.ID]
				cartCouponsFixed          = make([]*coupons.Redemption, 0)
				cartCouponsPercentage     = make([]*coupons.Redemption, 0)
				cartItemCouponsFixed      = make(map[drivers.ULID][]*coupons.Redemption)
				cartItemCouponsPercentage = make(map[drivers.ULID][]*coupons.Redemption)
			)

			// Group Redemptions
			for _, redemption := range redemptions {
				cartRow.Coupons = append(cartRow.Coupons, redemption)

				var productsList []queries.RelO2O[*products.Product, *coupons.CouponProductThrough]
				var productsNil = redemption.Coupon.Products == nil
				if !productsNil {
					productsList = redemption.Coupon.Products.AsList()
				}

				if productsNil || len(productsList) == 0 {
					if redemption.Coupon.IsPercentage {
						cartCouponsPercentage = append(cartCouponsPercentage, redemption)
					} else {
						cartCouponsFixed = append(cartCouponsFixed, redemption)
					}
					continue
				}

				couponProducts := redemption.Coupon.Products.AsList()
				var m map[drivers.ULID][]*coupons.Redemption
				if redemption.Coupon.IsPercentage {
					m = cartItemCouponsPercentage
				} else {
					m = cartItemCouponsFixed
				}

				for _, product := range couponProducts {
					l, ok := m[product.Object.ID]
					if !ok {
						l = make([]*coupons.Redemption, 0)
					}
					m[product.Object.ID] = append(l, redemption)
				}
			}

			// Sort Fixed Coupons highest-to-lowest discount to apply biggest discounts first
			sort.Slice(cartCouponsFixed, func(i, j int) bool {
				return cartCouponsFixed[i].Coupon.MaxDiscount.GreaterThan(cartCouponsFixed[j].Coupon.MaxDiscount)
			})

			cartTotals := price.Totals{
				TotalPrice:       decimal.Zero,
				TotalFee:         decimal.Zero,
				TotalTax:         decimal.Zero,
				TotalDiscount:    decimal.Zero,
				LeftOverDiscount: decimal.Zero,
			}

			// To process cart-wide fixed discounts as a cascading pool later
			var totalCartWideFixedPool decimal.Decimal = decimal.Zero
			for _, cp := range cartCouponsFixed {
				totalCartWideFixedPool = totalCartWideFixedPool.Add(cp.Coupon.MaxDiscount)
			}

			// Process Items
			for _, item := range items {
				productID := item.Product.ID

				// Build the base price structure
				p := price.Price{
					ItemPrice:     item.Product.Price,
					ItemFee:       price.AmountValue{Amount: item.Product.AdditionalFee, IsPercentage: false},
					TaxPercentage: item.Product.TaxRate,
				}

				// Aggregate item-specific fixed discounts (Highest first)
				var itemFixedDiscountPool decimal.Decimal = decimal.Zero
				if itemFixedCoupons, exists := cartItemCouponsFixed[productID]; exists {
					sort.Slice(itemFixedCoupons, func(i, j int) bool {
						return itemFixedCoupons[i].Coupon.MaxDiscount.GreaterThan(itemFixedCoupons[j].Coupon.MaxDiscount)
					})
					for _, cp := range itemFixedCoupons {
						itemFixedDiscountPool = itemFixedDiscountPool.Add(cp.Coupon.MaxDiscount)
					}
				}

				// Apply cart-wide fixed discounts if we still have pool left
				if totalCartWideFixedPool.GreaterThan(decimal.Zero) {
					itemFixedDiscountPool = itemFixedDiscountPool.Add(totalCartWideFixedPool)
					// Note: Realistically, cart-wide pool logic requires updating the pool based on what was used,
					// which gets complex with `price.Calculate`. The `price` package handles leftovers per-item,
					// so we aggregate all fixed into the calculation.
				}

				// Apply fixed discounts FIRST
				_, calculatedPrice := p.Calculate(price.PriceCalculation{
					ItemCount: int(item.Amount),
					Discount:  price.AmountValue{Amount: itemFixedDiscountPool, IsPercentage: false},
				})

				// If the item isn't free yet, apply percentage discounts (Benefits business: applied AFTER fixed)
				// Aggregate percentage discounts
				var totalPercentage decimal.Decimal = decimal.Zero
				if itemPctCoupons, exists := cartItemCouponsPercentage[productID]; exists {
					for _, cp := range itemPctCoupons {
						totalPercentage = totalPercentage.Add(cp.Coupon.MaxDiscount) // e.g. "20" for 20%
					}
				}
				for _, cp := range cartCouponsPercentage {
					totalPercentage = totalPercentage.Add(cp.Coupon.MaxDiscount)
				}

				if totalPercentage.GreaterThan(decimal.Zero) {
					// To correctly apply percentage on the REMAINING price, we create a new Price struct
					// representing the post-fixed-discount state.
					remainingPrice := price.Price{
						ItemPrice:     calculatedPrice.TotalPrice.Div(decimal.NewFromInt(int64(item.Amount))), // Per item price
						ItemFee:       price.AmountValue{Amount: calculatedPrice.TotalFee.Div(decimal.NewFromInt(int64(item.Amount))), IsPercentage: false},
						TaxPercentage: item.Product.TaxRate,
					}

					_, pctCalculated := remainingPrice.Calculate(price.PriceCalculation{
						ItemCount: int(item.Amount),
						Discount:  price.AmountValue{Amount: totalPercentage, IsPercentage: true},
					})

					// Combine the discounts
					calculatedPrice.TotalPrice = pctCalculated.TotalPrice
					calculatedPrice.TotalFee = pctCalculated.TotalFee
					calculatedPrice.TotalTax = pctCalculated.TotalTax
					calculatedPrice.TotalDiscount = calculatedPrice.TotalDiscount.Add(pctCalculated.TotalDiscount)
				}

				// Deduct from the global fixed pool what this item actually used from it
				if calculatedPrice.LeftOverDiscount.GreaterThan(decimal.Zero) {
					// The item couldn't absorb all fixed discount.
					// If the leftover is greater than or equal to the cart-wide pool, it means the item-specific
					// discount covered the item, and the cart-wide pool remains untouched.
					if calculatedPrice.LeftOverDiscount.GreaterThanOrEqual(totalCartWideFixedPool) {
						// Cart pool is untouched, leftover is just unused item-specific discount
					} else {
						// Cart pool was partially used. The new pool is whatever is leftover.
						totalCartWideFixedPool = calculatedPrice.LeftOverDiscount
					}
				} else {
					// The item absorbed all fixed discounts, including the cart-wide pool.
					totalCartWideFixedPool = decimal.Zero
				}

				// Add to cart totals
				cartTotals.TotalPrice = cartTotals.TotalPrice.Add(calculatedPrice.TotalPrice)
				cartTotals.TotalFee = cartTotals.TotalFee.Add(calculatedPrice.TotalFee)
				cartTotals.TotalTax = cartTotals.TotalTax.Add(calculatedPrice.TotalTax)
				cartTotals.TotalDiscount = cartTotals.TotalDiscount.Add(calculatedPrice.TotalDiscount)

				// Assemble the specific item row
				itemRedemptions := make([]*coupons.Redemption, 0)
				if r, ok := cartItemCouponsFixed[productID]; ok {
					itemRedemptions = append(itemRedemptions, r...)
				}
				if r, ok := cartItemCouponsPercentage[productID]; ok {
					itemRedemptions = append(itemRedemptions, r...)
				}

				cartRow.Items = append(cartRow.Items, &CartItemRow{
					CartItem:        item,
					Redemptions:     itemRedemptions,
					CalculatedPrice: calculatedPrice,
				})
			}

			cartRow.Totals = cartTotals

			if !yield(cartRow, nil) {
				return
			}
		}
	}

	return n, seq, nil
}
