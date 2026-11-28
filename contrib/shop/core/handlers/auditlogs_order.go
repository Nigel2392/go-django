package handlers

import (
	"net/http"

	auditlogs "github.com/Nigel2392/go-django/contrib/reports/audit_logs"
	"github.com/Nigel2392/go-django/contrib/shop/models/orders"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/trans"
)

type orderCreatedDefinition struct{}

func (orderCreatedDefinition) TypeLabel(r *http.Request, _ string) string {
	return trans.T(r.Context(), "Order created")
}

func (d orderCreatedDefinition) GetLabel(r *http.Request, entry auditlogs.LogEntry) string {
	return d.TypeLabel(r, entry.Type())
}

func (orderCreatedDefinition) FormatMessage(r *http.Request, entry auditlogs.LogEntry) any {
	return trans.T(r.Context(), "Order %v was created with %v lines.",
		entry.ObjectID(), entry.Data()["line_count"])
}

func (orderCreatedDefinition) GetActions(r *http.Request, entry auditlogs.LogEntry) []auditlogs.LogEntryAction {
	return orderAuditActions(r, entry.ObjectID())
}

type orderUpdatedDefinition struct{}

func (orderUpdatedDefinition) TypeLabel(r *http.Request, _ string) string {
	return trans.T(r.Context(), "Order updated")
}

func (d orderUpdatedDefinition) GetLabel(r *http.Request, entry auditlogs.LogEntry) string {
	return d.TypeLabel(r, entry.Type())
}

func (orderUpdatedDefinition) FormatMessage(r *http.Request, entry auditlogs.LogEntry) any {
	return trans.T(r.Context(), "Order %v was updated. State %v, lines %v.",
		entry.ObjectID(), orderAuditState(r, entry), entry.Data()["line_count"])
}

func (orderUpdatedDefinition) GetActions(r *http.Request, entry auditlogs.LogEntry) []auditlogs.LogEntryAction {
	return orderAuditActions(r, entry.ObjectID())
}

type orderAcceptedDefinition struct{}

func (orderAcceptedDefinition) TypeLabel(r *http.Request, _ string) string {
	return trans.T(r.Context(), "Order accepted")
}

func (d orderAcceptedDefinition) GetLabel(r *http.Request, entry auditlogs.LogEntry) string {
	return d.TypeLabel(r, entry.Type())
}

func (orderAcceptedDefinition) FormatMessage(r *http.Request, entry auditlogs.LogEntry) any {
	return trans.T(r.Context(), "Order %v was accepted with %v lines.",
		entry.ObjectID(), entry.Data()["line_count"])
}

func (orderAcceptedDefinition) GetActions(r *http.Request, entry auditlogs.LogEntry) []auditlogs.LogEntryAction {
	return orderAuditActions(r, entry.ObjectID())
}

type orderProcessedDefinition struct{}

func (orderProcessedDefinition) TypeLabel(r *http.Request, _ string) string {
	return trans.T(r.Context(), "Order processed")
}

func (d orderProcessedDefinition) GetLabel(r *http.Request, entry auditlogs.LogEntry) string {
	return d.TypeLabel(r, entry.Type())
}

func (orderProcessedDefinition) FormatMessage(r *http.Request, entry auditlogs.LogEntry) any {
	return trans.T(r.Context(), "Order %v finished processing.",
		entry.ObjectID())
}

func (orderProcessedDefinition) GetActions(r *http.Request, entry auditlogs.LogEntry) []auditlogs.LogEntryAction {
	return orderAuditActions(r, entry.ObjectID())
}

type orderCompletedDefinition struct{}

func (orderCompletedDefinition) TypeLabel(r *http.Request, _ string) string {
	return trans.T(r.Context(), "Order completed")
}

func (d orderCompletedDefinition) GetLabel(r *http.Request, entry auditlogs.LogEntry) string {
	return d.TypeLabel(r, entry.Type())
}

func (orderCompletedDefinition) FormatMessage(r *http.Request, entry auditlogs.LogEntry) any {
	return trans.T(r.Context(), "Order %v was completed.",
		entry.ObjectID())
}

func (orderCompletedDefinition) GetActions(r *http.Request, entry auditlogs.LogEntry) []auditlogs.LogEntryAction {
	return orderAuditActions(r, entry.ObjectID())
}

func orderAuditState(r *http.Request, entry auditlogs.LogEntry) string {
	return orders.InternalOrderState(attrs.ToString(entry.Data()["state"])).Label(r.Context())
}

func orderAuditActions(r *http.Request, id any) []auditlogs.LogEntryAction {
	var orderID drivers.ULID
	if err := orderID.UnmarshalText([]byte(attrs.ToString(id))); err != nil || orderID.IsZero() {
		return nil
	}
	return shopEditAuditAction(r, &orders.Order{ID: orderID}, "orders", orderID, trans.T(r.Context(), "Edit order"), "orders:edit")
}
