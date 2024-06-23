package auditlogs

import (
	"fmt"
	"time"

	"github.com/Nigel2392/django/core/contenttypes"
	"github.com/Nigel2392/django/core/logger"
	"github.com/google/uuid"
)

type Entry struct {
	Id    uuid.UUID                `json:"id"`
	Typ   string                   `json:"type"`
	Lvl   logger.LogLevel          `json:"level"`
	Time  time.Time                `json:"timestamp"`
	UsrID interface{}              `json:"user_id"`
	Obj   interface{}              `json:"-"`
	ObjID interface{}              `json:"object_id"`
	CType contenttypes.ContentType `json:"content_type"`
	Src   map[string]interface{}   `json:"data"`
}

func (l *Entry) String() string {
	var (
		id      = l.ID()
		typ     = l.Type()
		objId   = l.ObjectID()
		cTyp    = l.ContentType()
		srcData = l.Data()
	)

	switch {
	case objId != nil && srcData != nil:
		return fmt.Sprintf(
			"<LogEntry(%q): %s> %s(%v) %v",
			typ, id, cTyp.TypeName(), objId, srcData,
		)
	case objId != nil:
		return fmt.Sprintf(
			"<LogEntry(%q): %s> %s(%v)",
			typ, id, cTyp.TypeName(), objId,
		)
	case srcData != nil:
		return fmt.Sprintf(
			"<LogEntry(%q): %s> %s %v",
			typ, id, cTyp.TypeName(), srcData,
		)
	}

	return fmt.Sprintf(
		"<LogEntry(%q): %s> %s",
		typ, id, cTyp.TypeName(),
	)
}

func (l *Entry) ID() uuid.UUID {
	return l.Id
}

func (l *Entry) Type() string {
	return l.Typ
}

func (l *Entry) Level() logger.LogLevel {
	return l.Lvl
}

func (l *Entry) Timestamp() time.Time {
	return l.Time
}

func (l *Entry) UserID() interface{} {
	return l.UsrID
}

func (l *Entry) Object() interface{} {
	return l.Obj
}

func (l *Entry) ObjectID() interface{} {
	return l.ObjID
}

func (l *Entry) ContentType() contenttypes.ContentType {
	return l.CType
}

func (l *Entry) Data() map[string]interface{} {
	return l.Src
}
