package contenttypes

import (
	"fmt"
	"strings"
)

var lazyRegistry = make(map[string]*LazyRegistry)

// A lazy registry for content types.
//
// This allows for lazy loading of content types, which can be useful in cases where
// the content types are not known at compile time or before the application is initialized.
//
// It is used to load content types by their type names, and it will panic if none of the provided type names are found in the registry.
//
// If no type names are provided, it will load the default model, which is the model that comes first in alphabetical order of the type names.
//
// This can be useful in many cases, for example in [./src/contrib/auth/users/baseuser.go] to load custom definitions of the user model,
// allowing for apps to not depend on a single user model.
//
// This type must be instantiated using [NewLazyRegistry] and passing a function that checks if the content type definition is valid for the lazy registry,
// such as checking if it implements a specific interface or has a specific field.
type LazyRegistry struct {
	models       map[string]*ContentTypeDefinition
	defaultModel *string
}

// NewLazyRegistry creates a new lazy registry for content types.
func NewLazyRegistry(modelKey string, check func(*ContentTypeDefinition) bool) *LazyRegistry {

	var (
		models       = make(map[string]*ContentTypeDefinition)
		defaultModel = new(string)
	)

	var _, err = OnRegister(func(def *ContentTypeDefinition) {
		if check(def) {
			var typeName = def.ContentType().ShortTypeName()

			// Set the first model as the default if not set or if the current type name is lexicographically smaller,
			// I.E. it comes before the current default in alphabetical order.
			if *defaultModel == "" || strings.Compare(*defaultModel, typeName) > 0 {
				*defaultModel = typeName
			}

			models[typeName] = def
		}
	})
	if err != nil {
		panic(fmt.Errorf("could not hook into contenttypes registry: %w", err))
	}

	var reg = &LazyRegistry{
		models:       models,
		defaultModel: defaultModel,
	}
	lazyRegistry[modelKey] = reg
	return reg
}

// Load loads a content type definition by its type name.
//
// If no type names are provided, it will load the default model.
func (r LazyRegistry) Load(typeNames ...string) *ContentTypeDefinition {
	if len(typeNames) == 0 {
		typeNames = []string{*r.defaultModel}
	}

	for _, typeName := range typeNames {
		var def = DefinitionForType(typeName)
		if def == nil {
			continue
		}

		typeName = def.ContentType().ShortTypeName()
		if def, ok := r.models[typeName]; ok {
			return def
		}
	}

	panic(fmt.Errorf(
		"LazyRegistry.Load(): called with unknown type name %q, available types: %v",
		strings.Join(typeNames, ", "), mapKeys(r.models),
	))
}

// LoadString loads a content type definition by its type name and returns its short type name.
func (r LazyRegistry) LoadString(typeNames ...string) string {
	if len(typeNames) == 0 {
		typeNames = []string{*r.defaultModel}
	}

	for _, typeName := range typeNames {
		var def = DefinitionForType(typeName)
		if def == nil {
			continue
		}

		typeName = def.ContentType().ShortTypeName()
		if _, ok := r.models[typeName]; ok {
			return typeName
		}
	}

	panic(fmt.Errorf(
		"LazyRegistry.Load(): called with unknown type name %q, available types: %v",
		strings.Join(typeNames, ", "), mapKeys(r.models),
	))
}

func mapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// LoadModel loads a content type definition by its type name and returns the model.
func LoadModel(modelKey string, typeNames ...string) *ContentTypeDefinition {
	var reg, ok = lazyRegistry[modelKey]
	if !ok {
		var ct = DefinitionForType(modelKey)
		if ct != nil {
			return ct
		}

		panic(fmt.Errorf(
			"LazyRegistry.LoadModel(): called with unknown model key %q in lazyRegistry with keys %v",
			modelKey, mapKeys(lazyRegistry)),
		)
	}

	return reg.Load(typeNames...)
}

// LoadModelString loads a content type definition by its type name and returns its short type name.
func LoadModelString(modelKey string, typeNames ...string) string {
	var reg, ok = lazyRegistry[modelKey]
	if !ok {
		panic(fmt.Errorf(
			"LazyRegistry.LoadModelString(): called with unknown model key %q in lazyRegistry with keys %v",
			modelKey, mapKeys(lazyRegistry)),
		)
	}

	return reg.LoadString(typeNames...)
}
