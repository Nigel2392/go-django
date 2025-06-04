package attrs

import (
	"fmt"
	"reflect"
)

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
		return typIndirected.FieldByName(name)
	}

	field, ok = structTyp.fields[name]
	if !ok {
		return typIndirected.FieldByName(name)
	}

	return field, true
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
