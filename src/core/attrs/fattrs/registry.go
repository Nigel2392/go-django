package fattrs

import (
	"fmt"
	"reflect"
)

type optionsType = any

var cachedOptions = make(map[reflect.Type]map[string]optionsType) // map for ReflectLessField options

func AddOptions(typ reflect.Type, name string, opts optionsType) {
	var optsMap, ok = cachedOptions[typ]
	if !ok {
		optsMap = make(map[string]optionsType)
		cachedOptions[typ] = optsMap
	}

	optsMap[name] = opts
}

func GetOptions[OPTIONS any](typ reflect.Type, name string) (opts OPTIONS, ok bool) {
	optsMap, ok := cachedOptions[typ]
	if !ok {
		optsMap = make(map[string]optionsType)
		cachedOptions[typ] = optsMap
	}

	optsV, ok := optsMap[name]
	if !ok {
		// panic(fmt.Sprintf("ReflectLessField options were not configured for field %q in type %s", name, typ))
		return opts, false
	}

	opts, ok = optsV.(OPTIONS)
	if !ok {
		panic(fmt.Sprintf("ReflectLessField options are not of type %T, got %T", opts, opts))
	}

	return opts, true
}
