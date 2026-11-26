package handlers

import (
	"net/http"

	auditlogs "github.com/Nigel2392/go-django/contrib/reports/audit_logs"
	"github.com/Nigel2392/go-django/contrib/shop/carts"
	"github.com/Nigel2392/go-django/src/core/trans"
)

type cartCreatedDefinition struct{}

func (cartCreatedDefinition) TypeLabel(r *http.Request, _ string) string {
	return trans.T(r.Context(), "Cart created")
}

func (d cartCreatedDefinition) GetLabel(r *http.Request, entry auditlogs.LogEntry) string {
	return d.TypeLabel(r, entry.Type())
}

func (cartCreatedDefinition) FormatMessage(r *http.Request, entry auditlogs.LogEntry) any {
	return trans.T(r.Context(), "Cart %v was created.",
		entry.ObjectID())
}

func (cartCreatedDefinition) GetActions(r *http.Request, entry auditlogs.LogEntry) []auditlogs.LogEntryAction {
	return cartAuditActions(r)
}

type cartClearedDefinition struct{}

func (cartClearedDefinition) TypeLabel(r *http.Request, _ string) string {
	return trans.T(r.Context(), "Cart cleared")
}

func (d cartClearedDefinition) GetLabel(r *http.Request, entry auditlogs.LogEntry) string {
	return d.TypeLabel(r, entry.Type())
}

func (cartClearedDefinition) FormatMessage(r *http.Request, entry auditlogs.LogEntry) any {
	return trans.T(r.Context(), "Cart %v was cleared.",
		entry.ObjectID())
}

func (cartClearedDefinition) GetActions(r *http.Request, entry auditlogs.LogEntry) []auditlogs.LogEntryAction {
	return cartAuditActions(r)
}

type cartItemAddedDefinition struct{}

func (cartItemAddedDefinition) TypeLabel(r *http.Request, _ string) string {
	return trans.T(r.Context(), "Cart item added")
}

func (d cartItemAddedDefinition) GetLabel(r *http.Request, entry auditlogs.LogEntry) string {
	return d.TypeLabel(r, entry.Type())
}

func (cartItemAddedDefinition) FormatMessage(r *http.Request, entry auditlogs.LogEntry) any {
	return trans.T(r.Context(), "Item %v was added to cart %v.",
		entry.ObjectID(), entry.Data()["cart_id"])
}

func (cartItemAddedDefinition) GetActions(r *http.Request, entry auditlogs.LogEntry) []auditlogs.LogEntryAction {
	return cartAuditActions(r)
}

type cartItemRemovedDefinition struct{}

func (cartItemRemovedDefinition) TypeLabel(r *http.Request, _ string) string {
	return trans.T(r.Context(), "Cart item removed")
}

func (d cartItemRemovedDefinition) GetLabel(r *http.Request, entry auditlogs.LogEntry) string {
	return d.TypeLabel(r, entry.Type())
}

func (cartItemRemovedDefinition) FormatMessage(r *http.Request, entry auditlogs.LogEntry) any {
	return trans.T(r.Context(), "Item %v was removed from cart %v.",
		entry.ObjectID(), entry.Data()["cart_id"])
}

func (cartItemRemovedDefinition) GetActions(r *http.Request, entry auditlogs.LogEntry) []auditlogs.LogEntryAction {
	return cartAuditActions(r)
}

func cartAuditActions(r *http.Request) []auditlogs.LogEntryAction {
	return shopListAuditAction(r, &carts.Cart{}, "carts", trans.T(r.Context(), "View carts"))
}
