package search

import (
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	django "github.com/Nigel2392/go-django/src"
)

type backendRegistry struct {
	defaultBackend string
	backends       map[string]SearchBackend
}

var registry = &backendRegistry{
	backends: make(map[string]SearchBackend),
}

func GetDefaultSearchBackend() (string, bool) {
	var v, ok = django.ConfigGetOK[string](
		django.Global.Settings,
		APPVAR_SEARCH_BACKEND,
	)
	return v, ok
}

func RegisterSearchBackend(name string, backend SearchBackend, setDefault ...bool) {
	if len(setDefault) > 0 && setDefault[0] {
		registry.defaultBackend = name
	}
	registry.backends[name] = backend
}

func GetSearchBackend(name string) (SearchBackend, bool) {
	var backend, ok = registry.backends[name]
	return backend, ok
}

func GetSearchBackendForModel[T SearchableModel](model T) (SearchBackend, error) {
	var backend SearchBackend
	if bd, ok := any(model).(BackendDefiner); ok {
		backend = bd.SearchBackend()
	} else {
		var backendName, ok = GetDefaultSearchBackend()
		if !ok && registry.defaultBackend != "" {
			backendName = registry.defaultBackend
			ok = true
		}
		if !ok {
			return nil, errors.NoDatabase.Wrapf(
				"No default search backend configured and model %T does not define one", model,
			)
		}

		backend, ok = GetSearchBackend(backendName)
		if !ok {
			return nil, errors.NoDatabase.Wrapf(
				"Search backend '%s' not registered",
				backendName,
			)
		}
	}

	return backend, nil
}
