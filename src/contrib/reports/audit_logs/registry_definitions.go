package auditlogs

import (
	"fmt"
	"html/template"
	"maps"
	"net/http"
	"slices"
	"strings"

	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/core/secrets/safety"
	"github.com/Nigel2392/go-django/src/core/trans"
)

var (
	_ Definition = (*loginFailedDefinition)(nil)
)

type loginFailedDefinition struct{}

func (p *loginFailedDefinition) TypeLabel(r *http.Request, typeName string) string {
	return trans.T(r.Context(), "Login failed")
}

func (p *loginFailedDefinition) GetLabel(r *http.Request, logEntry LogEntry) string {
	var data = logEntry.Data()
	var ip = data["ip"].(string)

	return trans.T(r.Context(), "Login failed from %s", ip)
}

func (p *loginFailedDefinition) GetActions(r *http.Request, l LogEntry) []LogEntryAction {
	return nil
}

func (p *loginFailedDefinition) FormatMessage(r *http.Request, logEntry LogEntry) any {
	var data = logEntry.Data()
	var ip = data["ip"].(string)
	var sb = new(strings.Builder)
	sb.WriteString("<p>")
	sb.WriteString(trans.T(r.Context(), "IP Address: %s", ip))
	sb.WriteString("</p>\n")

	var form, ok = data["form"].(map[string]interface{})
	if !ok {
		logger.Warn("Expected form field to be a map, got %T", data["form"])
		return template.HTML(sb.String())
	}

	for k := range form {
		if safety.IsSecretField(r.Context(), k) {
			delete(form, k)
		}
	}

	var formKeys = slices.Collect(maps.Keys(form))
	slices.Sort(formKeys)

	if len(formKeys) == 0 {
		return template.HTML(sb.String())
	}

	sb.WriteString("<p>")
	sb.WriteString(trans.T(r.Context(), "Form data:"))
	sb.WriteString("</p>\n")
	sb.WriteString("<ul>\n")
	for _, k := range formKeys {
		var val, ok = form[k].([]interface{})
		if !ok {
			logger.Warn("Expected form field '%s' to be an array, got %T", k, form[k])
			continue
		}

		var v any = val
		switch len(val) {
		case 0:
			v = "<<empty>>"
		case 1:
			v = val[0]
		default:
			v = val
		}

		fmt.Fprintf(sb, "<li>%s: %v</li>\n", k, v)
	}
	sb.WriteString("</ul>\n")

	return template.HTML(sb.String())
}
