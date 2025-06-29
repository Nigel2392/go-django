package auditlogs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/google/uuid"
)

var (
	ErrLogEntryNotFound = errors.New("log entry not found")
	ErrLogsNotReady     = errors.New("audit logs app not ready")

	LogUnknownTypes bool = false
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
	GetLabel(r *http.Request, logEntry LogEntry) string
	FormatMessage(r *http.Request, logEntry LogEntry) string
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

	var entry = &Entry{
		Typ:  drivers.String(entryType),
		Lvl:  level,
		Time: drivers.CurrentTimestamp(),
		Src:  drivers.JSON[map[string]any]{Data: data},
	}

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

	err = entry.Save(ctx)
	return entry.ID(), err
}
