package handlers

import (
	"net/http"
	"net/url"

	auditlogs "github.com/Nigel2392/go-django/contrib/reports/audit_logs"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/permissions"
)

func shopEditAuditAction(r *http.Request, object attrs.Definer, model string, id any, label string, extraPermissions ...string) []auditlogs.LogEntryAction {
	if django.Global == nil || !django.AppInstalled("shop") || !permissions.HasObjectPermission(r, object, "admin:edit") ||
		!permissions.HasObjectPermission(r, object, extraPermissions...) {
		return nil
	}
	query := url.Values{"next": {r.URL.Path}}
	return []auditlogs.LogEntryAction{&auditlogs.BaseAction{
		DisplayLabel: label,
		ActionURL:    django.Reverse("admin:apps:model:edit", "shop", model, id) + "?" + query.Encode(),
	}}
}

func shopListAuditAction(r *http.Request, object attrs.Definer, model, label string) []auditlogs.LogEntryAction {
	if django.Global == nil || !django.AppInstalled("shop") || !permissions.HasObjectPermission(r, object, "admin:view_list") {
		return nil
	}
	return []auditlogs.LogEntryAction{&auditlogs.BaseAction{
		DisplayLabel: label,
		ActionURL:    django.Reverse("admin:apps:model", "shop", model),
	}}
}
