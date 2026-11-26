package orders

import "github.com/Nigel2392/go-django/contrib/shop/shared/signals"

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
