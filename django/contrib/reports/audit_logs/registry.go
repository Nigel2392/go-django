package auditlogs

import (
	"bytes"
	"fmt"
	"time"

	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/core/contenttypes"
	"github.com/Nigel2392/django/core/logger"
	"github.com/google/uuid"
)

type auditLogRegistry struct {
	// filters and handlers are maps of maps of slices to allow for multiple filters and handlers of the same type
	// to be registered
	// The first key is the type of the filter/handler
	// The second key is the content type of the object the filter/handler should be applied to
	filtersCtyp  map[string]map[string][]Filter
	handlersCtyp map[string]map[string][]Handler

	// filters and handlers are maps of slices to allow for multiple filters and handlers of the same type
	// these handlers do NOT have a content type associated with them
	filters  map[string][]Filter
	handlers map[string][]Handler

	// Backend is used to store and retrieve log entries
	backend StorageBackend
}

var registry = &auditLogRegistry{
	filtersCtyp:  make(map[string]map[string][]Filter),
	handlersCtyp: make(map[string]map[string][]Handler),
	filters:      make(map[string][]Filter),
	handlers:     make(map[string][]Handler),
	backend:      newInMemoryStorageBackend(),
}

func Backend() StorageBackend {
	return registry.backend
}

func RegisterBackend(backend StorageBackend) {
	registry.backend = backend
}

func RegisterFilter(filter Filter) {
	var typ = filter.Type()
	if registry.filters[typ] == nil {
		registry.filters[typ] = make([]Filter, 0)
	}
	registry.filters[typ] = append(registry.filters[typ], filter)
}

func RegisterHandler(handler Handler) {
	var typ = handler.Type()
	if registry.handlers[typ] == nil {
		registry.handlers[typ] = make([]Handler, 0)
	}
	registry.handlers[typ] = append(registry.handlers[typ], handler)
}

func RegisterFilterForObject(filter Filter, contentType contenttypes.ContentType) {
	var typ = filter.Type()
	var pkgPath = contentType.PkgPath()
	var m map[string][]Filter
	if registry.filtersCtyp[typ] == nil {
		m = make(map[string][]Filter)
	} else {
		m = registry.filtersCtyp[typ]
	}

	if m[pkgPath] == nil {
		m[pkgPath] = make([]Filter, 0)
	}

	m[pkgPath] = append(m[pkgPath], filter)
	registry.filtersCtyp[typ] = m
}

func RegisterHandlerForObject(handler Handler, contentType contenttypes.ContentType) {
	var typ = handler.Type()
	var pkgPath = contentType.PkgPath()
	var m map[string][]Handler
	if registry.handlersCtyp[typ] == nil {
		m = make(map[string][]Handler)
	} else {
		m = registry.handlersCtyp[typ]
	}

	if m[pkgPath] == nil {
		m[pkgPath] = make([]Handler, 0)
	}

	m[pkgPath] = append(m[pkgPath], handler)
	registry.handlersCtyp[typ] = m
}

func Log(entryType string, level logger.LogLevel, forObject attrs.Definer, data map[string]interface{}) (uuid.UUID, error) {
	var entry = &Entry{
		Typ:  entryType,
		Lvl:  level,
		Time: time.Now(),
		Src:  data,
	}

	var (
		filtersForTyp  []Filter
		handlersForTyp []Handler
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
			filtersForTyp = make([]Filter, 0)
		}

		handlersForTyp, ok = handlersMap[pkgPath]
		if !ok {
			handlersForTyp = make([]Handler, 0)
		}

	} else {
		filtersForTyp = make([]Filter, 0)
		handlersForTyp = make([]Handler, 0)
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
		var ok bool
		e, ok = filter.Filter(e)
		if !ok {
			e = entry
			break
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
