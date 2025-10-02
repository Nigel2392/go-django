package attrutils

import (
	"reflect"

	_ "unsafe"
)

var cachedStructs = make(reflectStructFieldMap) // Cache for struct fields and methods

func GetStructField(typIndirected reflect.Type, name string) (*reflect.StructField, bool) {
	var f, ok, _ = cachedStructs.getField(typIndirected, name)
	return f, ok
}

func GetStructMethod(typIndirected reflect.Type, name string) (reflect.Method, bool) {
	return cachedStructs.getMethod(typIndirected, name)
}

func AddStructField(typIndirected reflect.Type, name string, field *reflect.StructField) {
	var (
		structTyp, ok = cachedStructs[typIndirected]
	)
	if !ok {
		structTyp = &structType{
			methods: make(map[string]reflect.Method),
			fields:  make(map[string]*reflect.StructField),
		}
		cachedStructs[typIndirected] = structTyp
	}

	structTyp.fields[name] = field
}

func AddStructMethod(typIndirected reflect.Type, name string, method reflect.Method) {
	var (
		structTyp, ok = cachedStructs[typIndirected]
	)
	if !ok {
		structTyp = &structType{
			methods: make(map[string]reflect.Method),
			fields:  make(map[string]*reflect.StructField),
		}
		cachedStructs[typIndirected] = structTyp
	}

	structTyp.methods[name] = method
}

type structType struct {
	methods map[string]reflect.Method
	fields  map[string]*reflect.StructField
}

type reflectStructFieldMap map[reflect.Type]*structType

func (m reflectStructFieldMap) getField(typIndirected reflect.Type, name string) (fld *reflect.StructField, found bool, cached bool) {
	cached = true
	var (
		structTyp, ok = m[typIndirected]
		field         *reflect.StructField
	)
	if !ok {
		structTyp = &structType{
			methods: make(map[string]reflect.Method),
			fields:  make(map[string]*reflect.StructField),
		}
		m[typIndirected] = structTyp
		cached = false
	}

	field, ok = structTyp.fields[name]
	if !ok {
		var fld reflect.StructField
		fld, ok = typIndirected.FieldByName(name)
		if ok {
			structTyp.fields[name] = &fld
			field = &fld
		}
		cached = false
	}

	return field, ok, cached
}

func (m reflectStructFieldMap) getMethod(typIndirected reflect.Type, name string) (reflect.Method, bool) {
	var (
		structTyp, ok = m[typIndirected]
		method        reflect.Method
	)

	if !ok {
		return method, false
	}

	method, ok = structTyp.methods[name]
	if !ok {
		return method, false
	}

	return method, true
}
