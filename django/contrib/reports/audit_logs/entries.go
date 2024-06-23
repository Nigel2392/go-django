package auditlogs

import (
	"time"

	"github.com/Nigel2392/django/core/contenttypes"
	"github.com/Nigel2392/django/core/logger"
	"github.com/google/uuid"
)

type logEntry struct {
	Id    uuid.UUID                `json:"id"`
	Typ   string                   `json:"type"`
	Lvl   logger.LogLevel          `json:"level"`
	Time  time.Time                `json:"timestamp"`
	Obj   interface{}              `json:"-"`
	ObjID interface{}              `json:"object_id"`
	CType contenttypes.ContentType `json:"content_type"`
	Src   map[string]interface{}   `json:"data"`
}

func (l *logEntry) ID() uuid.UUID {
	return l.Id
}

func (l *logEntry) Type() string {
	return l.Typ
}

func (l *logEntry) Level() logger.LogLevel {
	return l.Lvl
}

func (l *logEntry) Timestamp() time.Time {
	return l.Time
}

func (l *logEntry) Object() interface{} {
	return l.Obj
}

func (l *logEntry) ObjectID() interface{} {
	return l.ObjID
}

func (l *logEntry) ContentType() contenttypes.ContentType {
	return l.CType
}

func (l *logEntry) Data() map[string]interface{} {
	return l.Src
}
