package handlers

import (
	"net/http"
	"strconv"

	auditlogs "github.com/Nigel2392/go-django/contrib/reports/audit_logs"
	"github.com/Nigel2392/go-django/contrib/shop/coupons"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/trans"
)

type couponCreatedDefinition struct{}

func (couponCreatedDefinition) TypeLabel(r *http.Request, _ string) string {
	return trans.T(r.Context(), "Coupon created")
}

func (d couponCreatedDefinition) GetLabel(r *http.Request, entry auditlogs.LogEntry) string {
	return d.TypeLabel(r, entry.Type())
}

func (couponCreatedDefinition) FormatMessage(r *http.Request, entry auditlogs.LogEntry) any {
	return trans.T(r.Context(), "Coupon %q (ID %v) was created.",
		entry.Data()["title"], entry.ObjectID())
}

func (couponCreatedDefinition) GetActions(r *http.Request, entry auditlogs.LogEntry) []auditlogs.LogEntryAction {
	return couponAuditActions(r, entry.ObjectID())
}

type couponUpdatedDefinition struct{}

func (couponUpdatedDefinition) TypeLabel(r *http.Request, _ string) string {
	return trans.T(r.Context(), "Coupon updated")
}

func (d couponUpdatedDefinition) GetLabel(r *http.Request, entry auditlogs.LogEntry) string {
	return d.TypeLabel(r, entry.Type())
}

func (couponUpdatedDefinition) FormatMessage(r *http.Request, entry auditlogs.LogEntry) any {
	return trans.T(r.Context(), "Coupon %q (ID %v) was updated.",
		entry.Data()["title"], entry.ObjectID())
}

func (couponUpdatedDefinition) GetActions(r *http.Request, entry auditlogs.LogEntry) []auditlogs.LogEntryAction {
	return couponAuditActions(r, entry.ObjectID())
}

type couponRedeemedDefinition struct{}

func (couponRedeemedDefinition) TypeLabel(r *http.Request, _ string) string {
	return trans.T(r.Context(), "Coupon redeemed")
}

func (d couponRedeemedDefinition) GetLabel(r *http.Request, entry auditlogs.LogEntry) string {
	return d.TypeLabel(r, entry.Type())
}

func (couponRedeemedDefinition) FormatMessage(r *http.Request, entry auditlogs.LogEntry) any {
	return trans.T(r.Context(), "Coupon %q (ID %v) was redeemed on cart %v.",
		entry.Data()["title"], entry.ObjectID(), entry.Data()["cart_id"])
}

func (couponRedeemedDefinition) GetActions(r *http.Request, entry auditlogs.LogEntry) []auditlogs.LogEntryAction {
	return couponAuditActions(r, entry.ObjectID())
}

func couponAuditActions(r *http.Request, id any) []auditlogs.LogEntryAction {
	couponID, err := strconv.ParseUint(attrs.ToString(id), 10, 64)
	if err != nil || couponID == 0 {
		return nil
	}
	return shopEditAuditAction(r, &coupons.Coupon{ID: couponID}, "coupons", couponID, trans.T(r.Context(), "Edit coupon"))
}
