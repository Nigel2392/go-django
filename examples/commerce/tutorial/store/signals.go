package store

import (
	"context"
	"fmt"

	"github.com/Nigel2392/go-django/contrib/shop/core/shared/signals"
	"github.com/Nigel2392/go-django/contrib/shop/core/util/paystate"
	"github.com/Nigel2392/go-django/contrib/shop/models/orders"
	"github.com/Nigel2392/go-django/contrib/shop/models/payments"
	queries "github.com/Nigel2392/go-django/queries/src"
)

type DeliveryConfirmedSignal struct {
	signals.BaseSignal
	Order *orders.Order
}

var DeliveryConfirmed = signals.Pool.Get[DeliveryConfirmedSignal]("tutorial.store.Delivery.Confirmed")

func registerSignals() error {
	ctx := context.Background()
	if _, err := orders.Signals.Created.Listen(ctx, reviewOrder); err != nil {
		return err
	}
	if _, err := payments.Signals.Start.Listen(ctx, func(ctx context.Context, _ signals.Signal[payments.PaymentStartedSignal], event payments.PaymentStartedSignal) error {
		return scheduleDeliveries(ctx, event.Payment)
	}); err != nil {
		return err
	}
	if _, err := payments.Signals.Update.Listen(ctx, func(ctx context.Context, _ signals.Signal[payments.PaymentUpdatedSignal], event payments.PaymentUpdatedSignal) error {
		return scheduleDeliveries(ctx, event.Payment)
	}); err != nil {
		return err
	}
	_, err := DeliveryConfirmed.Listen(ctx, finishOrder)
	return err
}

func reviewOrder(ctx context.Context, _ signals.Signal[orders.OrderCreatedSignal], event orders.OrderCreatedSignal) error {
	// Make the decision from the saved product snapshots in the checkout transaction.
	decision := orders.OrderStateAccepted
review:
	for _, line := range event.Lines {
		product, ok := line.Product.Data.Specific.(*Product)
		if !ok || product == nil {
			return fmt.Errorf("order line %s has no tutorial product snapshot", line.ID)
		}
		switch product.Availability {
		case inStock:
		case discontinued:
			decision = orders.OrderStateRejected
			break review
		case backordered:
			decision = orders.OrderStatePending
		default:
			return fmt.Errorf("unknown product availability %q", product.Availability)
		}
	}
	if event.Order.State == decision {
		return nil
	}
	event.Order.State = decision
	if _, err := orders.Query().WithContext(ctx).Filter("ID", event.Order.ID).
		Select("State").Update(event.Order); err != nil {
		return err
	}
	if decision == orders.OrderStateRejected {
		return orders.ReconcileCouponRedemptions(ctx, event.Order.ID)
	}
	return nil
}

func scheduleDeliveries(ctx context.Context, payment *payments.Payment) error {
	if payment.State != paystate.STATUS_PAID && !payment.IsFreeCheckout() {
		return nil
	}
	order, ok := payment.Order.(*orders.Order)
	if !ok || order == nil || order.ID.IsZero() {
		return fmt.Errorf("payment %s has no order", payment.ID)
	}

	// The payment signals already own a transaction. A free checkout uses
	// this same handler after checkout returns, so it starts a transaction here.
	ctx, tx, err := queries.StartTransaction(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	row, err := orders.Query().WithContext(ctx).Filter("ID", order.ID).Get()
	if err != nil {
		return err
	}
	order = row.Object
	if order.Processed || (order.State != orders.OrderStateAccepted && order.State != orders.OrderStatePending) {
		return nil
	}

	lines, err := queries.GetQuerySet(&orders.OrderItem{}).WithContext(ctx).Filter("Order", order.ID).All()
	if err != nil {
		return err
	}
	// One job per line. Repeated payment notifications reuse the existing jobs.
	for _, line := range lines {
		if _, _, err := queries.GetQuerySet(&Delivery{}).WithContext(ctx).Filter("Line", line.Object.ID).
			GetOrCreate(&Delivery{Line: line.Object}); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func finishOrder(ctx context.Context, _ signals.Signal[DeliveryConfirmedSignal], event DeliveryConfirmedSignal) error {
	remaining, err := queries.GetQuerySet(&Delivery{}).WithContext(ctx).
		Filter("Line.Order", event.Order.ID).Filter("Receipt", "").Count()
	if err != nil || remaining > 0 {
		return err
	}
	// The built-in workflow calls each supplier to verify its stored receipt.
	// It saves completion only after every OrderAccepted callback succeeds.
	return orders.Signals.Completed.Send(ctx, orders.OrderCompletedSignal{
		BaseSignal: event.BaseSignal,
		Order:      event.Order,
	})
}
