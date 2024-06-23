package auditlogs

import (
	"fmt"
	"strings"
)

type BoundDefinition struct {
	Definition Definition
	LogEntry
}

func (bd BoundDefinition) Label() string {
	return bd.Definition.GetLabel(bd.LogEntry)
}

func (bd BoundDefinition) Message() string {
	return bd.Definition.FormatMessage(bd.LogEntry)
}

func (bd BoundDefinition) Actions() []LogEntryAction {
	return bd.Definition.GetActions(bd.LogEntry)
}

func SimpleDefinition(l LogEntry) Definition {
	return &simpleDefinition{}
}

type simpleDefinition struct{}

func (sd *simpleDefinition) GetLabel(logEntry LogEntry) string {
	var (
		id  = logEntry.ID()
		typ = logEntry.Type()
		lvl = logEntry.Level().String()
	)

	var b = new(strings.Builder)
	b.WriteString("[")
	b.WriteString(lvl)
	b.WriteString("] ")
	b.WriteString(typ)
	b.WriteString(" (")
	b.WriteString(id.String())
	b.WriteString(")")
	return b.String()
}

func (sd *simpleDefinition) FormatMessage(logEntry LogEntry) string {
	var (
		cType = logEntry.ContentType()
		objId = logEntry.ObjectID()
		src   = logEntry.Data()
	)

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

func (sd *simpleDefinition) GetActions(logEntry LogEntry) []LogEntryAction {
	return nil
}
