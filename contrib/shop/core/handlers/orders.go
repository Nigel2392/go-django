package handlers

import (
	"github.com/Nigel2392/go-django/contrib/shop/models/orders"
	"github.com/Nigel2392/go-django/src/core/logger"
)

func registerOrderAudits() {
	listenAudit(orders.Signals.Created, "shop:orders:create", orderCreatedDefinition{}, func(s orders.OrderCreatedSignal) auditEvent {
		return orderLinesAudit(s.Order, s.Lines)
	})
	listenAudit(orders.Signals.Updated, "shop:orders:update", orderUpdatedDefinition{}, func(s orders.OrderUpdatedSignal) auditEvent {
		return orderLinesAudit(s.Order, s.Lines)
	})
	listenAudit(orders.Signals.Accepted, "shop:orders:accept", orderAcceptedDefinition{}, func(s orders.OrderAcceptedSignal) auditEvent {
		return orderLinesAudit(s.Order, s.Lines)
	})
	listenAudit(orders.Signals.Processed, "shop:orders:process", orderProcessedDefinition{}, func(s orders.OrderProcessedSignal) auditEvent {
		return orderAudit(s.Order)
	})
	listenAudit(orders.Signals.Completed, "shop:orders:complete", orderCompletedDefinition{}, func(s orders.OrderCompletedSignal) auditEvent {
		return orderAudit(s.Order)
	})
}

func orderAudit(order *orders.Order) auditEvent {
	data := make(map[string]any)
	if order != nil {
		data["state"] = order.State
	}
	return auditEvent{object: order, level: logger.INF, data: data}
}

func orderLinesAudit(order *orders.Order, lines []*orders.OrderItem) auditEvent {
	entry := orderAudit(order)
	entry.data["line_count"] = len(lines)
	return entry
}
