package auditlogs

import (
	"fmt"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"

	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/core/trans"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type BoundDefinition struct {
	Request    *http.Request
	Definition Definition
	LogEntry
}

func (bd BoundDefinition) Label() string {
	return bd.Definition.GetLabel(bd.Request, bd.LogEntry)
}

func (bd BoundDefinition) Message() any {
	return bd.Definition.FormatMessage(bd.Request, bd.LogEntry)
}

func (bd BoundDefinition) Actions() []LogEntryAction {
	var actions = bd.Definition.GetActions(bd.Request, bd.LogEntry)
	if actions == nil {
		actions = make([]LogEntryAction, 0)
	}

	var (
		typ   = bd.LogEntry.Type()
		cType = bd.LogEntry.ContentType()
		objId = bd.LogEntry.ObjectID()
	)

	actions = append(actions, &BaseAction{
		DisplayLabel: trans.T(bd.Request.Context(), "This type only"),
		ActionURL:    fmt.Sprintf("?filters-type=%s", url.QueryEscape(typ)),
	})

	if !attrs.IsZero(cType) {
		actions = append(actions, &BaseAction{
			DisplayLabel: trans.T(bd.Request.Context(), "This model type only"),
			ActionURL:    fmt.Sprintf("?filters-content_type=%s", url.QueryEscape(cType.ShortTypeName())),
		})
	}

	if !attrs.IsZero(cType) && !attrs.IsZero(objId) {
		actions = append(actions, &BaseAction{
			DisplayLabel: trans.T(bd.Request.Context(), "This object only"),
			ActionURL: fmt.Sprintf(
				"?filters-object_id=%s&filters-content_type=%s",
				url.QueryEscape(attrs.ToString(objId)),
				url.QueryEscape(cType.ShortTypeName()),
			),
		})
	}

	return actions
}

func SimpleDefinition() Definition {
	return &simpleDefinition{}
}

type simpleDefinition struct{}

func (sd *simpleDefinition) TypeLabel(request *http.Request, typeName string) string {
	// return text.(strings.ReplaceAll(typeName, "_", " "))
	var labelParts = strings.FieldsFunc(typeName, func(r rune) bool {
		switch r {
		case '_', ':', '-', '.', ';', '/':
			return true
		}
		return false
	})
	var caser = cases.Title(language.English)
	return caser.String(strings.Join(labelParts, " "))
}

func (sd *simpleDefinition) GetLabel(request *http.Request, logEntry LogEntry) string {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("panic in GetLabel for log entry type %q: %v:\n%s", logEntry.Type(), r, debug.Stack())
		}
	}()

	var (
		typ   = logEntry.Type()
		cType = logEntry.ContentType()
		objId = logEntry.ObjectID()
	)

	var b = new(strings.Builder)
	b.WriteString(typ)
	if !attrs.IsZero(cType) {
		b.WriteString(" (")
		b.WriteString(cType.ShortTypeName())

		if !attrs.IsZero(objId) {
			b.WriteString(" ")
			b.WriteString(attrs.ToString(objId))
		}
		b.WriteString(")")
	}
	return b.String()
}

func (sd *simpleDefinition) FormatMessage(request *http.Request, logEntry LogEntry) any {
	var (
		cType = logEntry.ContentType()
		objId = logEntry.ObjectID()
		src   = logEntry.Data()
	)

	if attrs.IsZero(cType) && attrs.IsZero(objId) {
		if src != nil {
			return fmt.Sprintf("%v", src)
		}
		return ""
	}

	switch {
	case objId != nil && src != nil:
		return fmt.Sprintf(
			"%s(%v) %v",
			cType.TypeName(), objId, src,
		)
	case objId != nil:
		return fmt.Sprintf(
			"%s(%v)",
			cType.TypeName(), objId,
		)
	case src != nil:
		return fmt.Sprintf(
			"%s %v",
			cType.TypeName(), src,
		)
	}

	return cType.TypeName()
}

func (sd *simpleDefinition) GetActions(request *http.Request, logEntry LogEntry) []LogEntryAction {
	return nil
}
