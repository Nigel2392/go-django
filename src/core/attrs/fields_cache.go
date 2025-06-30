package attrs

import (
	"fmt"
	"reflect"

	_ "unsafe"
)

var cachedStructs = make(reflectStructFieldMap) // Cache for struct fields and methods

// add a global function so internal packages can linkname to it
//
//go:linkname getStructField
func getStructField(typ reflect.Type, name string) (reflect.StructField, bool) {
	// Indirect the type to get the underlying type
	typIndirected := typ
	if typ.Kind() == reflect.Ptr {
		typIndirected = typ.Elem()
	}

	return cachedStructs.getField(typIndirected, name)
}

type structType struct {
	methods map[string]reflect.Method
	fields  map[string]reflect.StructField
}

func nameGetDefault(f *FieldDef) string {
	return fmt.Sprintf("GetDefault%s", f.field_t.Name)
}

func nameSetValue(f *FieldDef) string {
	return fmt.Sprintf("Set%s", f.field_t.Name)
}

func nameGetValue(f *FieldDef) string {
	return fmt.Sprintf("Get%s", f.field_t.Name)
}

type reflectStructFieldMap map[reflect.Type]*structType

func (m reflectStructFieldMap) getField(typIndirected reflect.Type, name string) (reflect.StructField, bool) {
	var (
		structTyp, ok = m[typIndirected]
		field         reflect.StructField
	)
	if !ok {
		structTyp = &structType{
			methods: make(map[string]reflect.Method),
			fields:  make(map[string]reflect.StructField),
		}
		m[typIndirected] = structTyp
	}

	field, ok = structTyp.fields[name]
	if !ok {
		field, ok = typIndirected.FieldByName(name)
		if ok {
			structTyp.fields[name] = field
		}
	}

	return field, ok
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
