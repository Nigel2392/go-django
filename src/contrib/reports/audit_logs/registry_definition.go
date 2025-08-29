package auditlogs

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/trans"
)

type BoundDefinition struct {
	Request    *http.Request
	Definition Definition
	LogEntry
}

func (bd BoundDefinition) Label() string {
	return bd.Definition.GetLabel(bd.Request, bd.LogEntry)
}

func (bd BoundDefinition) Message() string {
	return bd.Definition.FormatMessage(bd.Request, bd.LogEntry)
}

func cloneQuery(query url.Values) url.Values {
	clone := make(url.Values, len(query))
	for k, v := range query {
		clone[k] = slices.Clone(v)
	}
	return clone
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

	if cType != nil {
		actions = append(actions, &BaseAction{
			DisplayLabel: trans.T(bd.Request.Context(), "This model type only"),
			ActionURL:    fmt.Sprintf("?filters-content_type=%s", url.QueryEscape(cType.ShortTypeName())),
		})
	}

	if cType != nil && !attrs.IsZero(objId) {
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

func (sd *simpleDefinition) GetLabel(request *http.Request, logEntry LogEntry) string {
	var (
		typ   = logEntry.Type()
		cType = logEntry.ContentType()
		objId = logEntry.ObjectID()
	)

	var b = new(strings.Builder)
	b.WriteString(typ)
	if cType != nil {
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

func (sd *simpleDefinition) FormatMessage(request *http.Request, logEntry LogEntry) string {
	return ""

	//var (
	//	cType = logEntry.ContentType()
	//	objId = logEntry.ObjectID()
	//	src   = logEntry.Data()
	//)
	//
	//switch {
	//case objId != nil && src != nil:
	//	return fmt.Sprintf(
	//		"%s(%v) %v",
	//		cType.TypeName(), objId, src,
	//	)
	//case objId != nil:
	//	return fmt.Sprintf(
	//		"%s(%v)",
	//		cType.TypeName(), objId,
	//	)
	//case src != nil:
	//	return fmt.Sprintf(
	//		"%s %v",
	//		cType.TypeName(), src,
	//	)
	//}
	//
	//return cType.TypeName()
}

func (sd *simpleDefinition) GetActions(request *http.Request, logEntry LogEntry) []LogEntryAction {
	return nil
}
