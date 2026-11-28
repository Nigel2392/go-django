package orders

import (
	"context"
	"uuid"

	"github.com/Nigel2392/go-django/contrib/shop/core/ledger"
	"github.com/Nigel2392/go-django/contrib/shop/core/ledger/sqledger"
	"github.com/Nigel2392/go-django/contrib/shop/core/shared/signals"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
)

// OrderCreatedSignal is sent within the order transaction by CreateOrder.
type OrderCreatedSignal struct {
	signals.BaseSignal
	Order *Order
	Lines []*OrderItem
}

type OrderUpdatedSignal struct {
	signals.BaseSignal
	Order *Order
	Lines []*OrderItem
}

type OrderAcceptedSignal struct {
	signals.BaseSignal
	Order *Order
	Lines []*OrderItem
}

type OrderProcessedSignal struct {
	signals.BaseSignal
	Order *Order
}

type OrderCompletedSignal struct {
	signals.BaseSignal
	Order *Order
}

type orderSignalManager struct {
	Created   signals.Signal[OrderCreatedSignal]
	Updated   signals.Signal[OrderUpdatedSignal]
	Accepted  signals.Signal[OrderAcceptedSignal]
	Processed signals.Signal[OrderProcessedSignal]
	Completed signals.Signal[OrderCompletedSignal]
}

var Signals = orderSignalManager{
	Created:   signals.Pool.Get[OrderCreatedSignal]("shop.orders.Order.Created"),
	Updated:   signals.Pool.Get[OrderUpdatedSignal]("shop.orders.Order.Updated"),
	Accepted:  signals.Pool.Get[OrderAcceptedSignal]("shop.orders.Order.Accepted"),
	Processed: signals.Pool.Get[OrderProcessedSignal]("shop.orders.Order.Processed"),
	Completed: signals.Pool.Get[OrderCompletedSignal]("shop.orders.Order.Completed"),
}

var Ledger = ledger.Ledgers.Add(func(u uuid.UUID, ctx context.Context) (ledger.Ledger[*OrderItem], error) {
	l := &sqledger.Ledger[*OrderItem]{
		Work: completeOrderItem,
	}

	return l, nil
})

func completeOrderItem(ctx context.Context, item *OrderItem, lease ledger.TaskState) (err error) {
	supplier, ok := RetrieveSupplier(item.Supplier)
	if !ok {
		return errors.Wrapf(ErrSupplierNotFound, "supplier %q is not found", item.Supplier)
	}

	itemRow, err := queries.
		GetQuerySet(&OrderItem{}).
		WithContext(ctx).
		Select("*", "Order.*", "Order.Customer.*").
		Filter("ID", item.ID).
		Get()
	if err != nil {
		return errors.Wrapf(err, "failed to query for order item %q", item.ID.String())
	}

	block, err := supplier.OrderBlock(ctx, itemRow.Object.Order, itemRow.Object)
	if err != nil {
		return errors.Wrapf(err, "failed to retrieve supplier block for %q", itemRow.Object.Supplier)
	}

	response, err := supplier.CompleteOrderItem(ctx, itemRow.Object.Order, itemRow.Object)
	if err != nil {
		return errors.Wrapf(err, "failed to accept item %q for supplier %q", item.ID.String(), itemRow.Object.Supplier)
	}
	if response == nil {
		return nil
	}

	err = response.BindToBlock(block)
	if err != nil {
		return errors.Wrapf(err, "failed to bind supplier block for item %q and supplier %q", item.ID.String(), itemRow.Object.Supplier)
	}

	itemRow.Object.SupplierData = response

	count, err := queries.
		GetQuerySet(&OrderItem{}).
		WithContext(ctx).
		Filter("ID", item.ID).
		Select("SupplierData").
		Update(itemRow.Object)
	if err != nil {
		return errors.Wrapf(err, "failed to save order item %q with supplier %q",
			item.ID.String(), itemRow.Object.Supplier,
		)
	}
	if count != 1 {
		return errors.NoChanges.Wrapf("failed to save supplier data for item %q", item.ID.String())
	}

	return nil
}
