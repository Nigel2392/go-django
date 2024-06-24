package auditlogs

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/core/contenttypes"
	"github.com/Nigel2392/django/core/logger"
	"github.com/google/uuid"
)

var (
	ErrLogEntryNotFound = errors.New("log entry not found")
	ErrLogsNotReady     = errors.New("audit logs app not ready")

	LogUnknownTypes bool = false
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

type EntryFilter interface {
	Type() string
	EntryFilter(message LogEntry) bool
}

type EntryHandler interface {
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

type StorageBackend interface {
	Setup() error
	Store(logEntry LogEntry) (uuid.UUID, error)
	StoreMany(logEntries []LogEntry) ([]uuid.UUID, error)
	Retrieve(id uuid.UUID) (LogEntry, error)
	RetrieveForObject(objectID interface{}, amount, offset int) ([]LogEntry, error)
	RetrieveForUser(userID interface{}, amount, offset int) ([]LogEntry, error)
	RetrieveMany(amount, offset int) ([]LogEntry, error)
	RetrieveTyped(logType string, amount, offset int) ([]LogEntry, error)
	EntryFilter(filters []AuditLogFilter, amount, offset int) ([]LogEntry, error)
}

func Log(entryType string, level logger.LogLevel, forObject attrs.Definer, data map[string]interface{}) (uuid.UUID, error) {

	if !Logs.IsReady() {
		return uuid.Nil, ErrLogsNotReady
	}

	if !LogUnknownTypes {
		if _, ok := registry.definitions[entryType]; !ok {
			err := fmt.Errorf(
				"no definition registered for log entry type %q, cannot log",
				entryType,
			)
			logger.Warn(err)
			return uuid.Nil, err
		}

	}

	var entry = &Entry{
		Typ:  entryType,
		Lvl:  level,
		Time: time.Now(),
		Src:  data,
	}

	var (
		filtersForTyp  []EntryFilter
		handlersForTyp []EntryHandler
		output         *bytes.Buffer = new(bytes.Buffer)
		e              LogEntry      = entry
		err            error
		ok             bool
	)

	if forObject != nil {
		var (
			contentType = contenttypes.NewContentType[interface{}](forObject)
			pkgPath     = contentType.PkgPath()
			filtersMap  = registry.filtersCtyp[entryType]
			handlersMap = registry.handlersCtyp[entryType]
			defs        = forObject.FieldDefs()
			primary     = defs.Primary()
		)

		entry.Obj = forObject
		entry.ObjID = primary.GetValue()
		entry.CType = contentType

		filtersForTyp, ok = filtersMap[pkgPath]
		if !ok {
			filtersForTyp = make([]EntryFilter, 0)
		}

		handlersForTyp, ok = handlersMap[pkgPath]
		if !ok {
			handlersForTyp = make([]EntryHandler, 0)
		}

	} else {
		filtersForTyp = make([]EntryFilter, 0)
		handlersForTyp = make([]EntryHandler, 0)
	}

	filters, ok := registry.filters[entryType]
	if ok {
		filtersForTyp = append(filtersForTyp, filters...)
	}

	handlers, ok := registry.handlers[entryType]
	if ok {
		handlersForTyp = append(handlersForTyp, handlers...)
	}

	logger.Logf(
		level, "Adding new %q entry to audit log", entryType,
	)

	if len(filtersForTyp) == 0 && len(handlersForTyp) == 0 {
		err = fmt.Errorf(
			"no filters or handlers registered for entry type %q",
			entryType,
		)

		logger.Warn(err)
		goto storeLogEntry
	}

	for _, filter := range filtersForTyp {
		if !filter.EntryFilter(e) {
			logger.Warnf("filter %q rejected log entry", filter.Type())
			return uuid.Nil, nil
		}
	}

	for _, handler := range handlersForTyp {
		if err := handler.Handle(output, e); err != nil {
			return uuid.Nil, err
		}
	}

storeLogEntry:
	if registry.backend != nil {
		return registry.backend.Store(e)
	}

	return uuid.Nil, err
}
