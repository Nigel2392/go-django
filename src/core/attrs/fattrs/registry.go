package fattrs

import (
	"fmt"
	"reflect"
	"strings"
)

type optionsType = any

var cachedOptions = make(map[reflect.Type]map[string]optionsType) // map for ReflectLessField options

func AddOptions(typ reflect.Type, name string, opts optionsType, extra ...string) {
	var optsMap, ok = cachedOptions[typ]
	if !ok {
		optsMap = make(map[string]optionsType)
		cachedOptions[typ] = optsMap
	}

	optsMap[optionsKey(name, extra)] = opts
}

func GetOptions[OPTIONS any](typ reflect.Type, name string, extra ...string) (opts OPTIONS, ok bool) {
	optsMap, ok := cachedOptions[typ]
	if !ok {
		optsMap = make(map[string]optionsType)
		cachedOptions[typ] = optsMap
	}

	optsV, ok := optsMap[optionsKey(name, extra)]
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

func optionsKey(name string, extra []string) string {
	if len(extra) == 0 {
		return name
	}

	var key strings.Builder
	key.WriteString(name)
	for _, part := range extra {
		key.WriteByte(':')
		key.WriteString(part)
	}
	return key.String()
}
