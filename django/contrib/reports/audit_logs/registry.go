package auditlogs

import (
	"net/http"

	"github.com/Nigel2392/django/core/contenttypes"
)

type auditLogRegistry struct {
	// filters and handlers are maps of maps of slices to allow for multiple filters and handlers of the same type
	// to be registered
	// The first key is the type of the filter/handler
	// The second key is the content type of the object the filter/handler should be applied to
	filtersCtyp  map[string]map[string][]EntryFilter
	handlersCtyp map[string]map[string][]EntryHandler

	// filters and handlers are maps of slices to allow for multiple filters and handlers of the same type
	// these handlers do NOT have a content type associated with them
	filters  map[string][]EntryFilter
	handlers map[string][]EntryHandler

	// Definitions are used for formatting the message to users or performing any additional operations on the log entry
	// before it is shown to the user
	definitions map[string]Definition

	// Backend is used to store and retrieve log entries
	backend StorageBackend
}

var registry = &auditLogRegistry{
	filtersCtyp:  make(map[string]map[string][]EntryFilter),
	handlersCtyp: make(map[string]map[string][]EntryHandler),
	filters:      make(map[string][]EntryFilter),
	handlers:     make(map[string][]EntryHandler),
	definitions:  make(map[string]Definition),
	backend:      NewInMemoryStorageBackend(),
}

func Backend() StorageBackend {
	return registry.backend
}

func RegisterBackend(backend StorageBackend) {
	registry.backend = backend
}

func RegisterFilter(filter EntryFilter) {
	var typ = filter.Type()
	if registry.filters[typ] == nil {
		registry.filters[typ] = make([]EntryFilter, 0)
	}
	registry.filters[typ] = append(registry.filters[typ], filter)
}

func RegisterHandler(handler EntryHandler) {
	var typ = handler.Type()
	if registry.handlers[typ] == nil {
		registry.handlers[typ] = make([]EntryHandler, 0)
	}
	registry.handlers[typ] = append(registry.handlers[typ], handler)
}

func RegisterFilterForObject(filter EntryFilter, contentType contenttypes.ContentType) {
	var typ = filter.Type()
	var pkgPath = contentType.PkgPath()
	var m map[string][]EntryFilter
	if registry.filtersCtyp[typ] == nil {
		m = make(map[string][]EntryFilter)
	} else {
		m = registry.filtersCtyp[typ]
	}

	if m[pkgPath] == nil {
		m[pkgPath] = make([]EntryFilter, 0)
	}

	m[pkgPath] = append(m[pkgPath], filter)
	registry.filtersCtyp[typ] = m
}

func RegisterHandlerForObject(handler EntryHandler, contentType contenttypes.ContentType) {
	var typ = handler.Type()
	var pkgPath = contentType.PkgPath()
	var m map[string][]EntryHandler
	if registry.handlersCtyp[typ] == nil {
		m = make(map[string][]EntryHandler)
	} else {
		m = registry.handlersCtyp[typ]
	}

	if m[pkgPath] == nil {
		m[pkgPath] = make([]EntryHandler, 0)
	}

	m[pkgPath] = append(m[pkgPath], handler)
	registry.handlersCtyp[typ] = m
}

func RegisterDefinition(typ string, definition Definition) {
	registry.definitions[typ] = definition
}

func Define(r *http.Request, l LogEntry) *BoundDefinition {
	var typ = l.Type()
	var def, ok = registry.definitions[typ]
	if !ok {
		def = SimpleDefinition()
	}

	return &BoundDefinition{
		Request:    r,
		Definition: def,
		LogEntry:   l,
	}
}
