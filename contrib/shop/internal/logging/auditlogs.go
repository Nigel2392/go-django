package logging

import (
	"fmt"
	"html"
	"html/template"
	"net/http"
	"slices"
	"strings"

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

		initial := trans.T(r.Context(), "%q with id %v was edited",
			label, logEntry.ObjectID(),
		)

		var sb strings.Builder
		sb.WriteString("<div class=\"comparison\">")
		sb.WriteString("<p>")
		sb.WriteString(html.EscapeString(initial))
		sb.WriteString("</p>")

		sb.WriteString("<ul>")
		var changes, _ = logEntry.Data()["changes"].([]any)
		for _, _change := range changes {
			change, ok := _change.(map[string]any)
			if !ok {
				continue
			}

			name, ok := change["name"].(string)
			if !ok {
				continue
			}

			htmlVal, ok := change["html"].(string)
			if !ok {
				continue
			}

			sb.WriteString("<li>")
			sb.WriteString("<strong>")
			sb.WriteString(html.EscapeString(name))
			sb.WriteString(":</strong>")
			sb.WriteString(htmlVal)
			sb.WriteString("</li>")
		}
		sb.WriteString("</ul>")
		sb.WriteString("</div>")

		return template.HTML(sb.String())
	}

	return trans.T(r.Context(), "Unknown shop log entry type")
}
