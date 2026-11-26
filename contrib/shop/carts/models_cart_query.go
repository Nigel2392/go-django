package carts

import (
	"cmp"
	"iter"
	"net/http"
	"slices"
	"strconv"
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
	"github.com/Nigel2392/mux/v2/middleware/authentication"
	"github.com/Nigel2392/mux/v2/middleware/sessions"
	"github.com/shopspring/decimal"
)

type CartQuery struct {
	*queries.WrappedQuerySet[*Cart, *CartQuery, *queries.QuerySet[*Cart]]
}

type CartItemRow struct {
	CartItem        *CartItem             `json:"item"`
	Redemptions     []*coupons.Redemption `json:"coupons"`
	CalculatedPrice price.Totals          `json:"total_price"`
	// RedemptionAmounts includes the positive amount applied by each redemption on this line.
	RedemptionAmounts map[uint64]decimal.Decimal `json:"-"`
}

type CartRow struct {
	Object  *Cart                 `json:"cart"`
	Items   []*CartItemRow        `json:"items"`
	Coupons []*coupons.Redemption `json:"coupons"`
	Totals  price.Totals          `json:"total_price"`
	// RedemptionAmounts includes the positive amount applied by each redemption across all lines.
	RedemptionAmounts map[uint64]decimal.Decimal `json:"-"`
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
	return qs.CartsForRequest(r).Filter("Active", true).OrderBy("CreatedAt")
}

// CartsForRequest scopes carts to the session or authenticated user,
// including inactive carts.
func (qs *CartQuery) CartsForRequest(r *http.Request) *CartQuery {
	if r == nil {
		return qs.
			Clone().
			Filter(expr.Q("Active", true).And(expr.Q("Active", false)))
	}

	var (
		user       = authentication.Retrieve(r)
		session    = sessions.Retrieve(r)
		hasUser    = user != nil && user.IsAuthenticated()
		hasSession = session != nil && session.Token() != ""
	)

	var baseQ expr.ClauseExpression
	switch {
	case hasSession && hasUser:
		baseQ = expr.Q("Session", session.Token()).Or(expr.Q("User", user))
	case hasSession:
		baseQ = expr.Q("Session", session.Token())
	case hasUser:
		baseQ = expr.Q("User", user)
	default:
		return qs.
			Clone().
			Filter(expr.Q("Active", true).And(expr.Q("Active", false)))
	}

	return qs.Clone().Filter(baseQ)
}

func (qs *CartQuery) PreloadProducts(productFields ...string) *CartQuery {
	var flds = make([]any, 0, len(productFields)+1)
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

// CalcOne returns the calculated totals for one cart.
func (qs *CartQuery) CalcOne() (*CartRow, error) {
	return qs.calcOne(false)
}

// CalcOneWithProducts loads complete products for callers that use the priced
// cart items to create order snapshots.
func (qs *CartQuery) CalcOneWithProducts() (*CartRow, error) {
	return qs.calcOne(true)
}

func (qs *CartQuery) calcOne(includeProductDetails bool) (row *CartRow, err error) {
	qs = qs.Clone()

	// limit to max_get_results
	qs = qs.Limit(queries.MAX_GET_RESULTS)

	guessSize, resIter, err := qs.calc(includeProductDetails)
	if err != nil {
		return row, errors.Wrap(err, "CalcOne/Calc")
	}

	results := make([]*CartRow, 0, guessSize)
	for row, err := range resIter {
		if err != nil {
			return row, errors.Wrap(err, "CalcOne/resIter")
		}

		results = append(results, row)
	}

	var resCnt = len(results)
	if resCnt == 0 {
		return row, errors.NoRows.Wrap(
			"CalcOne: no rows found for cart",
		)
	}

	if resCnt > 1 {
		var errResCnt string
		if queries.MAX_GET_RESULTS == 0 || resCnt < queries.MAX_GET_RESULTS {
			errResCnt = strconv.Itoa(resCnt)
		} else {
			errResCnt = strconv.Itoa(queries.MAX_GET_RESULTS-1) + "+"
		}

		return row, errors.MultipleRows.Wrapf(
			"CalcOne: multiple rows returned for cart: %s rows", errResCnt,
		)
	}

	return results[0], nil
}

func (qs *CartQuery) CalcOneOrCreate(create *Cart) (row *CartRow, created bool, err error) {
	// If the queryset is already in a transaction, that transaction will be used
	// automatically.
	tx, err := qs.GetOrCreateTransaction()
	if err != nil {
		return nil, false, errors.FailedStartTransaction.WithCause(err)
	}
	defer tx.Rollback(qs.Context())

	// Check if the object already exists
	row, err = qs.CalcOne()
	if err != nil {
		if errors.Is(err, errors.NoRows) {
			goto create
		} else {
			return nil, false, errors.Wrap(
				err, "failed to get cart",
			)
		}
	}

	// Object already exists, return it and commit the transaction
	if row != nil {
		return row, false, tx.Commit(qs.Context())
	}

	// Object does not exist, create it
create:
	obj, err := qs.Create(create)
	// obj, err := qs.BulkCreate([]T{value})
	if err != nil {
		return nil, false, errors.Wrap(
			err, "failed to create cart",
		)
	}

	row = &CartRow{
		Object: obj,
	}

	// Object was created successfully, commit the transaction
	return row, true, tx.Commit(qs.Context())
}

func (qs *CartQuery) Calc() (int, iter.Seq2[*CartRow, error], error) {
	return qs.calc(false)
}

func (qs *CartQuery) calc(includeProductDetails bool) (int, iter.Seq2[*CartRow, error], error) {
	ctx, tx, err := queries.StartTransaction(qs.Context())
	if err != nil {
		return 0, nil, errors.Wrap(err, "CartQuery.Calc[StartTransaction]")
	}

	defer tx.Rollback(ctx)

	qs = qs.Clone().WithContext(ctx)

	productFields := []any{"*", "Product.ID", "Product.Price", "Product.AdditionalFee", "Product.TaxRate"}
	if includeProductDetails {
		productFields = []any{"*", "Product.*"}
	}
	qs = qs.Preload(queries.Preload{
		Path:    "CartItems",
		NoJoins: true,
		QuerySet: queries.
			GetQuerySet[attrs.Definer](&CartItem{}).
			Select(productFields...).
			OrderBy("Product.Price", "ID"),
	})

	n, cartRows, err := qs.IterAll()
	if err != nil {
		return 0, nil, errors.Wrap(err, "CartQuery.Calc[IterAll]")
	}

	var cartRedemptions map[drivers.ULID][]*coupons.Redemption
	var ri iter.Seq2[*queries.Row[*coupons.Redemption], error]
	if n > 0 {
		redemptionsQs := queries.
			GetQuerySet(&coupons.Redemption{}).
			WithContext(ctx).
			Select("*", "Coupon.*").
			Filter(
				expr.Q("Invalidated", false),
				expr.Q("Cart__in", qs.Select("ID")),
				expr.Q("Order__isnull", true),
			).
			Preload("Coupon.Products")

		_, ri, err = redemptionsQs.IterAll()
		if err != nil {
			return 0, nil, errors.Wrap(err, "CartQuery.Calc[coupons.Redemptions/IterAll]")
		}

		cartRedemptions = make(map[drivers.ULID][]*coupons.Redemption)
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
	}

	seq := func(yield func(*CartRow, error) bool) {
		for row, err := range cartRows {
			if err != nil {
				yield(nil, errors.Wrap(err, "CartQuery.Calc[IterAll/Rows]"))
				return
			}

			items := row.Object.CartItems.AsList()
			cartRow := &CartRow{
				Object:            row.Object,
				Items:             make([]*CartItemRow, 0, len(items)),
				Coupons:           make([]*coupons.Redemption, 0),
				RedemptionAmounts: make(map[uint64]decimal.Decimal),
			}

			cartProductIDs := make(map[drivers.ULID]struct{}, len(items))
			for _, item := range items {
				if item.Product != nil {
					cartProductIDs[item.Product.ID] = struct{}{}
				}
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
				if redemption.Coupon == nil || !redemption.Coupon.MaxDiscount.IsPositive() {
					continue
				}

				var productsList []queries.RelO2O[*products.Product, *coupons.CouponProductThrough]
				if redemption.Coupon.Products != nil {
					productsList = redemption.Coupon.Products.AsList()
				}

				if len(productsList) == 0 {
					cartRow.Coupons = append(cartRow.Coupons, redemption)
					if redemption.Coupon.IsPercentage {
						cartCouponsPercentage = append(cartCouponsPercentage, redemption)
					} else {
						cartCouponsFixed = append(cartCouponsFixed, redemption)
					}
					continue
				}

				var m map[drivers.ULID][]*coupons.Redemption
				if redemption.Coupon.IsPercentage {
					m = cartItemCouponsPercentage
				} else {
					m = cartItemCouponsFixed
				}

				matched := false
				seenProducts := make(map[drivers.ULID]struct{}, len(productsList))
				for _, product := range productsList {
					if product.Object == nil {
						continue
					}
					pid := product.Object.ID
					if _, inCart := cartProductIDs[pid]; !inCart {
						continue
					}
					if _, dup := seenProducts[pid]; dup {
						continue
					}
					seenProducts[pid] = struct{}{}
					matched = true
					m[pid] = append(m[pid], redemption)
				}
				if matched {
					cartRow.Coupons = append(cartRow.Coupons, redemption)
				}
			}

			// Within each group, apply larger discounts first and break ties by redemption ID.
			slices.SortFunc(cartCouponsFixed, compareRedemptionPriority)
			slices.SortFunc(cartCouponsPercentage, compareRedemptionPriority)
			for _, group := range cartItemCouponsFixed {
				slices.SortFunc(group, compareRedemptionPriority)
			}
			for _, group := range cartItemCouponsPercentage {
				slices.SortFunc(group, compareRedemptionPriority)
			}

			cartTotals := price.Totals{
				TotalPrice:       decimal.Zero,
				TotalFee:         decimal.Zero,
				TotalTax:         decimal.Zero,
				TotalDiscount:    decimal.Zero,
				LeftOverDiscount: decimal.Zero,
			}

			// Keep a separate remaining balance for each fixed cart-wide coupon.
			cartFixedBalance := make(map[uint64]decimal.Decimal, len(cartCouponsFixed))
			totalCartFixedBalance := decimal.Zero
			for _, redemption := range cartCouponsFixed {
				cartFixedBalance[redemption.ID] = redemption.Coupon.MaxDiscount
				totalCartFixedBalance = totalCartFixedBalance.Add(redemption.Coupon.MaxDiscount)
			}

			hundred := decimal.NewFromInt(100)
			for _, item := range items {
				if item.Product == nil {
					continue
				}

				productID := item.Product.ID

				p := price.Price{
					ItemPrice:     item.Product.Price,
					ItemFee:       price.AmountValue{Amount: item.Product.AdditionalFee, IsPercentage: false},
					TaxPercentage: item.Product.TaxRate,
				}

				fixedCoupons := cartItemCouponsFixed[productID]
				percentageCoupons := cartItemCouponsPercentage[productID]
				fixedBudget := totalCartFixedBalance
				for _, redemption := range fixedCoupons {
					fixedBudget = fixedBudget.Add(redemption.Coupon.MaxDiscount)
				}

				_, calculatedPrice := p.Calculate(price.PriceCalculation{
					ItemCount: int(item.Amount),
					Discount: price.AmountValue{
						Amount: fixedBudget,
					},
				})

				amounts := make(map[uint64]decimal.Decimal)
				unallocated := calculatedPrice.TotalDiscount
				// Product-specific discounts take priority. Cart-wide fixed discounts consume
				// their remaining balances in order of ascending item price.
				for _, redemption := range fixedCoupons {
					if !unallocated.IsPositive() {
						break
					}
					applied := decimal.Min(unallocated, redemption.Coupon.MaxDiscount)
					amounts[redemption.ID] = applied
					unallocated = unallocated.Sub(applied)
				}
				for _, redemption := range cartCouponsFixed {
					if !unallocated.IsPositive() {
						break
					}
					balance := cartFixedBalance[redemption.ID]
					applied := decimal.Min(unallocated, balance)
					if !applied.IsPositive() {
						continue
					}
					amounts[redemption.ID] = applied
					unallocated = unallocated.Sub(applied)
					cartFixedBalance[redemption.ID] = balance.Sub(applied)
					totalCartFixedBalance = totalCartFixedBalance.Sub(applied)
				}

				// Percentage discounts share the remaining product price and the 100% cap.
				// Use cumulative amounts so the allocations add up even if division rounds.
				percentage := decimal.Zero
				percentageDiscount := decimal.Zero
				if item.Amount > 0 {
					for _, group := range [][]*coupons.Redemption{percentageCoupons, cartCouponsPercentage} {
						for _, redemption := range group {
							if percentage.Equal(hundred) {
								break
							}
							percentage = decimal.Min(hundred, percentage.Add(redemption.Coupon.MaxDiscount))
							discount := calculatedPrice.TotalPrice.Mul(percentage).Div(hundred)
							applied := discount.Sub(percentageDiscount)
							if applied.IsPositive() {
								amounts[redemption.ID] = applied
							}
							percentageDiscount = discount
						}
					}
				}
				if percentage.IsPositive() {
					// Apply percentage over the post-fixed-discount line total with ItemCount: 1 to prevent division by 0 or rounding drift.
					remainingPrice := price.Price{
						ItemPrice:     calculatedPrice.TotalPrice,
						ItemFee:       price.AmountValue{Amount: calculatedPrice.TotalFee, IsPercentage: false},
						TaxPercentage: item.Product.TaxRate,
					}

					_, pctCalculated := remainingPrice.Calculate(price.PriceCalculation{
						ItemCount: 1,
						Discount: price.AmountValue{
							Amount:       percentage,
							IsPercentage: true,
						},
					})

					// Combine the discounts
					calculatedPrice.TotalPrice = pctCalculated.TotalPrice
					calculatedPrice.TotalFee = pctCalculated.TotalFee
					calculatedPrice.TotalTax = pctCalculated.TotalTax
					calculatedPrice.TotalDiscount = calculatedPrice.TotalDiscount.Add(pctCalculated.TotalDiscount)
				}

				for id, amount := range amounts {
					cartRow.RedemptionAmounts[id] = cartRow.RedemptionAmounts[id].Add(amount)
				}

				// Add to cart totals
				cartTotals.TotalPrice = cartTotals.TotalPrice.Add(calculatedPrice.TotalPrice)
				cartTotals.TotalFee = cartTotals.TotalFee.Add(calculatedPrice.TotalFee)
				cartTotals.TotalTax = cartTotals.TotalTax.Add(calculatedPrice.TotalTax)
				cartTotals.TotalDiscount = cartTotals.TotalDiscount.Add(calculatedPrice.TotalDiscount)

				itemRedemptions := make([]*coupons.Redemption, 0)
				itemRedemptions = append(itemRedemptions, fixedCoupons...)
				itemRedemptions = append(itemRedemptions, percentageCoupons...)

				cartRow.Items = append(cartRow.Items, &CartItemRow{
					CartItem:          item,
					Redemptions:       itemRedemptions,
					CalculatedPrice:   calculatedPrice,
					RedemptionAmounts: amounts,
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

func compareRedemptionPriority(a, b *coupons.Redemption) int {
	if largerFirst := b.Coupon.MaxDiscount.Cmp(a.Coupon.MaxDiscount); largerFirst != 0 {
		return largerFirst
	}
	return cmp.Compare(a.ID, b.ID)
}
