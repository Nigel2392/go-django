package handlers

import (
	"net/http"

	auditlogs "github.com/Nigel2392/go-django/contrib/reports/audit_logs"
	"github.com/Nigel2392/go-django/contrib/shop/models/payments"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/trans"
)

type paymentStartedDefinition struct{}

func (paymentStartedDefinition) TypeLabel(r *http.Request, _ string) string {
	return trans.T(r.Context(), "Payment started")
}

func (d paymentStartedDefinition) GetLabel(r *http.Request, entry auditlogs.LogEntry) string {
	return d.TypeLabel(r, entry.Type())
}

func (paymentStartedDefinition) FormatMessage(r *http.Request, entry auditlogs.LogEntry) any {
	return trans.T(r.Context(), "Payment %v was started for %v %v. Status %v.",
		entry.ObjectID(), entry.Data()["amount"], entry.Data()["currency"], entry.Data()["state"])
}

func (paymentStartedDefinition) GetActions(r *http.Request, entry auditlogs.LogEntry) []auditlogs.LogEntryAction {
	return paymentAuditActions(r, entry)
}

type paymentUpdatedDefinition struct{}

func (paymentUpdatedDefinition) TypeLabel(r *http.Request, _ string) string {
	return trans.T(r.Context(), "Payment status updated")
}

func (d paymentUpdatedDefinition) GetLabel(r *http.Request, entry auditlogs.LogEntry) string {
	return d.TypeLabel(r, entry.Type())
}

func (paymentUpdatedDefinition) FormatMessage(r *http.Request, entry auditlogs.LogEntry) any {
	previous := attrs.ToString(entry.Data()["previous_state"])
	current := attrs.ToString(entry.Data()["state"])
	if previous == current {
		return trans.T(r.Context(), "Payment %v was updated while in status %v.", entry.ObjectID(), current)
	}
	return trans.T(r.Context(), "Payment %v changed from %v to %v.",
		entry.ObjectID(), previous, current)
}

func (paymentUpdatedDefinition) GetActions(r *http.Request, entry auditlogs.LogEntry) []auditlogs.LogEntryAction {
	return paymentAuditActions(r, entry)
}

type paymentRefundedDefinition struct{}

func (paymentRefundedDefinition) TypeLabel(r *http.Request, _ string) string {
	return trans.T(r.Context(), "Payment refunded")
}

func (d paymentRefundedDefinition) GetLabel(r *http.Request, entry auditlogs.LogEntry) string {
	return d.TypeLabel(r, entry.Type())
}

func (paymentRefundedDefinition) FormatMessage(r *http.Request, entry auditlogs.LogEntry) any {
	return trans.T(r.Context(), "Payment %v was refunded by %v %v for request %v.",
		entry.ObjectID(), entry.Data()["refund_amount"], entry.Data()["refund_currency"], entry.Data()["request_id"])
}

func (paymentRefundedDefinition) GetActions(r *http.Request, entry auditlogs.LogEntry) []auditlogs.LogEntryAction {
	return paymentAuditActions(r, entry)
}

type paymentCancelledDefinition struct{}

func (paymentCancelledDefinition) TypeLabel(r *http.Request, _ string) string {
	return trans.T(r.Context(), "Payment cancelled")
}

func (d paymentCancelledDefinition) GetLabel(r *http.Request, entry auditlogs.LogEntry) string {
	return d.TypeLabel(r, entry.Type())
}

func (paymentCancelledDefinition) FormatMessage(r *http.Request, entry auditlogs.LogEntry) any {
	return trans.T(r.Context(), "Payment %v was cancelled for request %v.",
		entry.ObjectID(), entry.Data()["request_id"])
}

func (paymentCancelledDefinition) GetActions(r *http.Request, entry auditlogs.LogEntry) []auditlogs.LogEntryAction {
	return paymentAuditActions(r, entry)
}

func paymentAuditActions(r *http.Request, entry auditlogs.LogEntry) []auditlogs.LogEntryAction {
	actions := orderAuditActions(r, entry.Data()["order_id"])
	return append(actions, shopListAuditAction(r, &payments.Payment{}, "payments", trans.T(r.Context(), "View payments"))...)
}
