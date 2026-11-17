package orders

import (
	"net/http"

	"github.com/Nigel2392/go-django/contrib/shop/carts"
	"github.com/Nigel2392/go-django/contrib/shop/customers"
	"github.com/Nigel2392/go-django/contrib/shop/products"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

func CreateOrder(r *http.Request, customer *customers.Customer, cart *carts.Cart, items ...*carts.CartItem) (*Order, []*OrderItem, error) {
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
			return nil, nil, errors.Wrap(err, "OrderQuery/CreateOrder/OrderItems/IterAll")
		}

		items = make([]*carts.CartItem, 0, n)
		for row, err := range itemsRows {
			if err != nil {
				return nil, nil, errors.Wrap(err, "OrderQuery/CreateOrder/OrderItems/itemsRows")
			}

			items = append(items, row.Object)
		}
	}

	var orderItems = make([]*OrderItem, 0, len(items))
	for _, cartItem := range items {
		if cartItem.Product == nil {
			return nil, nil, errors.NilPointer.Wrapf(
				"product in CartItem %s is nil", cartItem.ID,
			)
		}

		orderItem := &OrderItem{
			Order:     order,
			ItemPrice: cartItem.Product.Price,
			Amount:    int(cartItem.Amount),
			Product: drivers.JSON[products.Product]{
				Data: *cartItem.Product,
			},
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
