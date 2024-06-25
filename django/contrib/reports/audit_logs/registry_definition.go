package auditlogs

import (
	"net/http"
	"strings"
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

func (bd BoundDefinition) Actions() []LogEntryAction {
	return bd.Definition.GetActions(bd.Request, bd.LogEntry)
}

func SimpleDefinition() Definition {
	return &simpleDefinition{}
}

type simpleDefinition struct{}

func (sd *simpleDefinition) GetLabel(request *http.Request, logEntry LogEntry) string {
	var (
		id  = logEntry.ID()
		typ = logEntry.Type()
	)

	var b = new(strings.Builder)
	b.WriteString(typ)
	b.WriteString(" (")
	b.WriteString(id.String())
	b.WriteString(")")
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
