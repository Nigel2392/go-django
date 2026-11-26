package orders

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
	"maps"
	"net/http"
	"reflect"
	"slices"

	"github.com/Nigel2392/go-django/contrib/shop/carts"
	"github.com/Nigel2392/go-django/contrib/shop/coupons"
	"github.com/Nigel2392/go-django/contrib/shop/customers"
	"github.com/Nigel2392/go-django/contrib/shop/products"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

var ErrCartRedemptionsChanged = stderrors.New("cart coupon redemptions changed during order creation")

func CreateOrder(r *http.Request, customer *customers.Customer, cart *carts.Cart, items ...*carts.CartItem) (*Order, []*OrderItem, error) {
	return createOrder(r, customer, cart, nil, items)
}

// CreateOrderFromCart saves the priced cart as an order, including the line
// totals and applied coupon redemptions. Use a row loaded by CalcOneWithProducts.
func CreateOrderFromCart(r *http.Request, customer *customers.Customer, cart *carts.CartRow) (*Order, []*OrderItem, error) {
	if cart == nil || cart.Object == nil || cart.Object.ID.IsZero() || len(cart.Items) == 0 {
		return nil, nil, errors.ValueError.Wrap("a priced cart with items is required")
	}
	items := make([]*carts.CartItem, len(cart.Items))
	for i, row := range cart.Items {
		if row == nil {
			return nil, nil, errors.NilPointer.Wrap("priced cart item is nil")
		}
		items[i] = row.CartItem
	}
	return createOrder(r, customer, cart.Object, cart, items)
}

func createOrder(r *http.Request, customer *customers.Customer, cart *carts.Cart, pricedCart *carts.CartRow, items []*carts.CartItem) (*Order, []*OrderItem, error) {
	if r == nil {
		return nil, nil, errors.NilPointer.Wrap("request is nil")
	}

	ctx, tx, err := queries.StartTransaction(r.Context())
	if err != nil {
		return nil, nil, errors.Wrap(err, "OrderQuery/CreateOrder/StartTx")
	}

	defer tx.Rollback(ctx)

	order := attrs.Setup[*Order](ctx, &Order{
		State:    orderStatePending,
		Customer: customer,
	})

	if len(items) == 0 {
		if cart == nil {
			return nil, nil, errors.NilPointer.Wrap("cart is nil")
		}

		items = cart.CartItems.AsList()
	}

	if len(items) == 0 {
		if cart.CartItems == nil {
			attrs.Define(ctx, cart).Get("CartItems")
		}

		n, itemsRows, err := cart.CartItems.
			Objects().
			WithContext(ctx).
			Select("*", "Product.*").
			IterAll()
		if err != nil {
			return nil, nil, errors.Wrap(err, "OrderQuery/CreateOrder/CartItems/IterAll")
		}

		items = make([]*carts.CartItem, 0, n)
		for row, err := range itemsRows {
			if err != nil {
				return nil, nil, errors.Wrap(err, "OrderQuery/CreateOrder/CartItems/itemsRows")
			}

			items = append(items, row.Object)
		}
	}

	models := make(map[string]products.ProductModel)
	var productIdMap = make(map[string][]any)
	var idConvMap = make(map[*products.Product]any)
	for _, cartItem := range items {
		if cartItem == nil {
			return nil, nil, errors.NilPointer.Wrap("cart item is nil")
		}
		if cartItem.Product == nil {
			return nil, nil, errors.NilPointer.Wrapf(
				"OrderQuery/CreateOrder/CartItems/RelatedProducts: related product is nil for cart %s", cartItem.ID)
		}

		if cartItem.Product.SpecificContentType == nil {
			return nil, nil, errors.NilPointer.Wrapf(
				"OrderQuery/CreateOrder/CartItems/RelatedProducts: related product has no specific content type for cart item %s", cartItem.ID)
		}

		contentType := cartItem.Product.SpecificContentType.TypeName()
		model := models[contentType]
		if model == nil {
			model, err = newSnapshotProduct(cartItem.Product)
			if err != nil {
				return nil, nil, err
			}
			models[contentType] = model
		}

		primary := attrs.GetModelMeta(model).Primary()
		pk := reflect.New(primary.Type())
		if err := json.Unmarshal([]byte(cartItem.Product.SpecificID), pk.Interface()); err != nil {
			return nil, nil, errors.CorruptedData.Wrapf("invalid specific product ID for cart item %s: %v", cartItem.ID, err)
		}
		id := pk.Elem().Interface()
		productIdMap[contentType] = append(productIdMap[contentType], id)
		idConvMap[cartItem.Product] = attrs.PrimaryKey(ctx, id)
	}

	var productModelMap = make(map[string]map[any]products.ProductModel)
	for contentType, productIds := range productIdMap {
		modelTyp := models[contentType]
		meta := attrs.GetModelMeta(modelTyp)
		primary := meta.Primary()
		_, iter, err := queries.
			GetQuerySet(modelTyp).
			WithContext(ctx).
			Filter(fmt.Sprintf("%s__in", primary.Name()), productIds).
			IterAll()
		if err != nil {
			return nil, nil, errors.Wrap(err, "OrderQuery/CreateOrder/CartItems/SpecificProducts")
		}

		productsMap := make(map[any]products.ProductModel)
		for row, err := range iter {
			if err != nil {
				return nil, nil, errors.Wrap(err, "OrderQuery/CreateOrder/CartItems/SpecificProducts/iterAll")
			}

			var defs = attrs.Define(ctx, row.Object)
			var primary = defs.Primary()
			productsMap[attrs.PrimaryKey(ctx, primary)] = row.Object
		}

		productModelMap[contentType] = productsMap
	}

	var orderItems = make([]*OrderItem, 0, len(items))
	for i, cartItem := range items {
		contentType := cartItem.Product.SpecificContentType.TypeName()
		pk := idConvMap[cartItem.Product]
		specific := productModelMap[contentType][pk]
		if specific == nil {
			return nil, nil, errors.NoRows.Wrapf("specific product %s %v is missing for cart item %s", contentType, pk, cartItem.ID)
		}

		product := *cartItem.Product
		orderItem := &OrderItem{
			Order:     order,
			ItemPrice: cartItem.Product.Price,
			TaxRate:   cartItem.Product.TaxRate,
			Amount:    int(cartItem.Amount),
			Supplier:  cartItem.Product.Supplier,
			Product: drivers.JSON[ProductData]{
				Data: ProductData{
					Product:  &product,
					Specific: specific,
				},
			},
		}

		if pricedCart != nil {
			line := pricedCart.Items[i]
			orderItem.Pricing.Data = &OrderItemPricing{
				Totals:            line.CalculatedPrice,
				RedemptionAmounts: maps.Clone(line.RedemptionAmounts),
			}
		}
		orderItems = append(orderItems, orderItem)
	}

	if len(items) == 0 {
		return nil, nil, errors.NoRows.Wrapf(
			"no items found for cart ID %s",
			cart.ID,
		)
	}

	order, err = queries.
		GetQuerySet(&Order{}).
		WithContext(ctx).
		Create(order)
	if err != nil {
		return nil, nil, errors.Wrap(err, "OrderQuery/CreateOrder")
	}

	orderItems, err = queries.
		GetQuerySet(&OrderItem{}).
		WithContext(ctx).
		BulkCreate(orderItems)
	if err != nil {
		return nil, nil, errors.Wrap(err, "OrderQuery/CreateOrder/OrderItems")
	}

	if pricedCart != nil && len(pricedCart.RedemptionAmounts) != 0 {
		redemptions := queries.
			GetQuerySet(&coupons.Redemption{}).
			WithContext(ctx)
		// Consistent update order prevents competing conversions from locking
		// the same reservations in different orders.
		for _, id := range slices.Sorted(maps.Keys(pricedCart.RedemptionAmounts)) {
			amount := pricedCart.RedemptionAmounts[id]
			count, err := redemptions.
				Filter(
					expr.Q("ID", id),
					expr.Q("Cart", cart.ID),
					expr.Q("Invalidated", false),
					expr.Q("Order__isnull", true),
				).
				Select("Order", "Customer", "DiscountAmount").
				Update(&coupons.Redemption{
					Order:          order,
					Customer:       customer,
					DiscountAmount: amount,
				})
			if err != nil {
				return nil, nil, errors.Wrap(err, "OrderQuery/CreateOrder/Redemptions")
			}
			if count != 1 {
				return nil, nil, ErrCartRedemptionsChanged
			}
		}
	}

	err = Signals.Created.Send(ctx, OrderCreatedSignal{
		Request: r.WithContext(ctx),
		Order:   order,
		Lines:   orderItems,
	})
	if err != nil {
		return nil, nil, errors.Wrap(err, "OrderQuery/CreateOrder/Signal")
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, nil, errors.Wrap(err, "OrderQuery/CreateOrder/TxCommit")
	}

	return order, orderItems, nil
}
