package orders

import "github.com/Nigel2392/go-django/contrib/shop/util/signals"

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

type orderSignalManager struct {
	Created signals.Signal[OrderCreatedSignal]
	Updated signals.Signal[OrderUpdatedSignal]
}

var Signals = orderSignalManager{
	Created: signals.Pool.Get[OrderCreatedSignal]("shop.orders.Order.Created"),
	Updated: signals.Pool.Get[OrderUpdatedSignal]("shop.orders.Order.Updated"),
}
