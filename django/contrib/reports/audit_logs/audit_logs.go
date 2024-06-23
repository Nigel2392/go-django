package auditlogs

import (
	"io"
	"net/http"
	"time"

	"github.com/Nigel2392/django/core/contenttypes"
	"github.com/Nigel2392/django/core/logger"
	"github.com/google/uuid"
)

type LogEntry interface {
	ID() uuid.UUID
	Type() string
	Level() logger.LogLevel
	Timestamp() time.Time
	UserID() interface{}
	ObjectID() interface{}
	ContentType() contenttypes.ContentType
	Data() map[string]interface{}
}

type Filter interface {
	Type() string
	Filter(message LogEntry) (possiblyNew LogEntry, ok bool)
}

type Handler interface {
	Type() string
	Handle(w io.Writer, message LogEntry) error
}

type LogEntryAction interface {
	Icon() string
	Label() string
	URL() string
}

type Definition interface {
	GetLabel(r *http.Request, logEntry LogEntry) string
	FormatMessage(r *http.Request, logEntry LogEntry) string
	GetActions(r *http.Request, logEntry LogEntry) []LogEntryAction
}
