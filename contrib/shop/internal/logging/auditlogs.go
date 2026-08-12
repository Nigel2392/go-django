package logging

import (
	"fmt"
	"net/http"
	"slices"

	auditlogs "github.com/Nigel2392/go-django/contrib/reports/audit_logs"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/permissions"
)

type shopLogDefinition struct {
	auditlogs.Definition
}

func NewShopLogDefinition() *shopLogDefinition {
	return &shopLogDefinition{
		Definition: auditlogs.SimpleDefinition(),
	}
}

func (p *shopLogDefinition) GetLabel(r *http.Request, logEntry auditlogs.LogEntry) string {
	var cType = logEntry.ContentType()
	var cTypeDef = contenttypes.DefinitionForType(cType.TypeName())
	var cTypeName = cType.ShortTypeName()
	if cType != nil {
		cTypeName = cTypeDef.Label(r.Context())

	}

	switch logEntry.Type() {
	case "shop:product:add":
		return trans.T(r.Context(), "%s was added",
			cTypeName,
		)

	case "shop:product:edit":
		return trans.T(r.Context(), "%s was changed",
			cTypeName,
		)
	}

	return trans.T(r.Context(), "Unknown shop log entry type")
}

func (p *shopLogDefinition) GetActions(r *http.Request, l auditlogs.LogEntry) []auditlogs.LogEntryAction {
	var id = l.ObjectID()
	if id == nil {
		return nil
	}

	var actions = make([]auditlogs.LogEntryAction, 0)
	if in(l.Type(), "shop:product:add", "shop:product:edit") && permissions.HasPermission(r, "shop:products:edit") {
		actions = append(actions, &auditlogs.BaseAction{
			DisplayLabel: trans.T(r.Context(), "Edit Product"),
			ActionURL: fmt.Sprintf("%s?%s=%s",
				django.Reverse("admin:shop:products:edit", id),
				"next",
				r.URL.Path,
			),
		})
	}

	return actions
}

func in(needle string, haystack ...string) bool {
	return slices.Contains(haystack, needle)
}

func (p *shopLogDefinition) FormatMessage(r *http.Request, logEntry auditlogs.LogEntry) any {
	var label, _ = logEntry.Data()["title"].(string)
	var cType = logEntry.ContentType()
	var cTypeDef = contenttypes.DefinitionForType(cType.TypeName())
	var cTypeName = cType.ShortTypeName()
	if cType != nil {
		cTypeName = cTypeDef.Label(r.Context())
	}

	if label == "" {
		label = cTypeName
	}

	switch logEntry.Type() {
	case "shop:product:add":
		return trans.T(r.Context(), "%q with id %v was added",
			label, logEntry.ObjectID(),
		)
	case "shop:product:edit":
		return trans.T(r.Context(), "%q with id %v was edited",
			label, logEntry.ObjectID(),
		)
	}

	return trans.T(r.Context(), "Unknown shop log entry type")
}
