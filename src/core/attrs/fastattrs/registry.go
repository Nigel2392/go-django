package fastattrs

import (
	"fmt"
	"reflect"

	"github.com/Nigel2392/go-django/src/core/attrs"
)

type optionsType = any

var cachedOptions = make(map[reflect.Type]map[string]optionsType) // map for ReflectLessField options

func AddOptions[T attrs.Definer](typ reflect.Type, name string, opts optionsType) {
	var optsMap, ok = cachedOptions[typ]
	if !ok {
		optsMap = make(map[string]optionsType)
		cachedOptions[typ] = optsMap
	}

	optsMap[name] = opts
}

func GetOptions[T attrs.Definer](typ reflect.Type, name string) (*reflectlessFieldOpts[T], bool) {
	optsMap, ok := cachedOptions[typ]
	if !ok {
		optsMap = make(map[string]optionsType)
		cachedOptions[typ] = optsMap
	}

	opts, ok := optsMap[name]
	if !ok {
		// panic(fmt.Sprintf("ReflectLessField options were not configured for field %q in type %s", name, typ))
		return nil, false
	}

	o, ok := opts.(*reflectlessFieldOpts[T])
	if !ok {
		panic(fmt.Sprintf("ReflectLessField options are not of type *reflectLessFieldOpts[T], got %T", opts))
	}

	return o, true
}
