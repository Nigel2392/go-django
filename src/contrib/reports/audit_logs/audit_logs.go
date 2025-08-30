package auditlogs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/contrib/auth/users"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/mux/middleware/authentication"
	"github.com/google/uuid"

	_ "unsafe"
)

var (
	ErrLogEntryNotFound = errors.New("log entry not found")
	ErrLogsNotReady     = errors.New("audit logs app not ready")

	LogUnknownTypes bool = true
)

type EntryFilter interface {
	Type() string
	EntryFilter(message *Entry) bool
}

type EntryHandler interface {
	Type() string
	Handle(w io.Writer, message *Entry) error
}

type LogEntryAction interface {
	Icon() string
	Label() string
	URL() string
}

type Definition interface {
	TypeLabel(r *http.Request, typeName string) string
	GetLabel(r *http.Request, logEntry LogEntry) string
	FormatMessage(r *http.Request, logEntry LogEntry) any // string | template.HTML
	GetActions(r *http.Request, logEntry LogEntry) []LogEntryAction
}

func Log(ctx context.Context, entryType string, level logger.LogLevel, forObject attrs.Definer, data map[string]interface{}) (uuid.UUID, error) {

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

	var user = authentication.UserFromContext(ctx).(users.User)
	var entry = models.Setup(&Entry{
		Typ:  drivers.String(entryType),
		Lvl:  level,
		Time: drivers.CurrentTimestamp(),
		Src:  drivers.JSON[map[string]any]{Data: data},
		Usr:  user,
	})

	var (
		filtersForTyp  []EntryFilter
		handlersForTyp []EntryHandler
		output         *bytes.Buffer = new(bytes.Buffer)
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
		entry.ObjID = drivers.JSON[any]{
			Data: primary.GetValue(),
		}
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

	if len(filtersForTyp) == 0 && len(handlersForTyp) == 0 {
		err = fmt.Errorf(
			"no filters or handlers registered for entry type %q",
			entryType,
		)

		logger.Warn(err)
		goto storeLogEntry
	}

	for _, filter := range filtersForTyp {
		if !filter.EntryFilter(entry) {
			logger.Warnf("filter %q rejected log entry", filter.Type())
			return uuid.Nil, nil
		}
	}

	for _, handler := range handlersForTyp {
		if err := handler.Handle(output, entry); err != nil {
			return uuid.Nil, err
		}
	}

storeLogEntry:
	logger.Logf(
		level, "Adding new %q entry to audit log", entryType,
	)

	if err = entry.Create(ctx); err != nil {
		logger.Errorf("Error saving log entry: %v", err)
		return uuid.Nil, err
	}

	return entry.ID(), err
}
