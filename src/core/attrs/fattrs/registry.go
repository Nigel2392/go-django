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

	var sb strings.Builder
	sb.WriteString(name)
	for _, opt := range extra {
		sb.WriteRune(':')
		sb.WriteString(opt)
	}

	optsMap[sb.String()] = opts
}

func GetOptions[OPTIONS any](typ reflect.Type, name string, extra ...string) (opts OPTIONS, ok bool) {
	optsMap, ok := cachedOptions[typ]
	if !ok {
		optsMap = make(map[string]optionsType)
		cachedOptions[typ] = optsMap
	}

	var sb strings.Builder
	sb.WriteString(name)
	for _, opt := range extra {
		sb.WriteRune(':')
		sb.WriteString(opt)
	}

	optsV, ok := optsMap[sb.String()]
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
