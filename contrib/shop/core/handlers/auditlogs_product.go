package handlers

import (
	"html"
	"html/template"
	"net/http"
	"strings"

	auditlogs "github.com/Nigel2392/go-django/contrib/reports/audit_logs"
	"github.com/Nigel2392/go-django/contrib/shop/models/products"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/trans"
)

type productAddedDefinition struct{}

func (productAddedDefinition) TypeLabel(r *http.Request, _ string) string {
	return trans.T(r.Context(), "Product added")
}

func (productAddedDefinition) GetLabel(r *http.Request, entry auditlogs.LogEntry) string {
	return trans.T(r.Context(), "%s was added", productAuditType(r, entry))
}

func (productAddedDefinition) FormatMessage(r *http.Request, entry auditlogs.LogEntry) any {
	return trans.T(r.Context(), "%q with id %v was added", productAuditTitle(r, entry), entry.ObjectID())
}

func (productAddedDefinition) GetActions(r *http.Request, entry auditlogs.LogEntry) []auditlogs.LogEntryAction {
	return productAuditActions(r, entry.ObjectID())
}

type productEditedDefinition struct{}

func (productEditedDefinition) TypeLabel(r *http.Request, _ string) string {
	return trans.T(r.Context(), "Product edited")
}

func (productEditedDefinition) GetLabel(r *http.Request, entry auditlogs.LogEntry) string {
	return trans.T(r.Context(), "%s was changed", productAuditType(r, entry))
}

func (productEditedDefinition) FormatMessage(r *http.Request, entry auditlogs.LogEntry) any {
	initial := trans.T(r.Context(), "%q with id %v was edited", productAuditTitle(r, entry), entry.ObjectID())
	var comparison strings.Builder
	comparison.WriteString("<div class=\"comparison\"><p>")
	comparison.WriteString(html.EscapeString(initial))
	comparison.WriteString("</p><ul>")

	changes, _ := entry.Data()["changes"].([]any)
	for _, rawChange := range changes {
		change, ok := rawChange.(map[string]any)
		if !ok {
			continue
		}
		name, hasName := change["name"].(string)
		rendered, hasHTML := change["html"].(string)
		if !hasName || !hasHTML {
			continue
		}
		comparison.WriteString("<li><strong>")
		comparison.WriteString(html.EscapeString(name))
		comparison.WriteString(":</strong>")
		comparison.WriteString(rendered)
		comparison.WriteString("</li>")
	}

	comparison.WriteString("</ul></div>")
	return template.HTML(comparison.String())
}

func (productEditedDefinition) GetActions(r *http.Request, entry auditlogs.LogEntry) []auditlogs.LogEntryAction {
	return productAuditActions(r, entry.ObjectID())
}

func productAuditType(r *http.Request, entry auditlogs.LogEntry) string {
	if model := entry.ContentType(); !attrs.IsZero(model) {
		if def := contenttypes.DefinitionForType(model.TypeName()); def != nil {
			return def.Label(r.Context())
		}
		return model.ShortTypeName()
	}
	return trans.T(r.Context(), "Product")
}

func productAuditTitle(r *http.Request, entry auditlogs.LogEntry) string {
	if title, _ := entry.Data()["title"].(string); title != "" {
		return title
	}
	return productAuditType(r, entry)
}

func productAuditActions(r *http.Request, id any) []auditlogs.LogEntryAction {
	var productID drivers.ULID
	if err := productID.UnmarshalText([]byte(attrs.ToString(id))); err != nil || productID.IsZero() {
		return nil
	}
	return shopEditAuditAction(r, &products.Product{ID: productID}, "products", productID, trans.T(r.Context(), "Edit Product"), "products:edit")
}
