package attrs

import (
	"fmt"
	"reflect"
)

type FieldDef struct {
	Blank          bool
	Editable       bool
	instance_t_ptr reflect.Type
	instance_v_ptr reflect.Value
	instance_t     reflect.Type
	instance_v     reflect.Value
	field_t        reflect.StructField
	field_v        reflect.Value
}

func NewField[T any](instance *T, name string, blank, editable bool) *FieldDef {
	var (
		instance_t_ptr = reflect.TypeOf(instance)
		instance_v_ptr = reflect.ValueOf(instance)
		instance_t     = instance_t_ptr.Elem()
		instance_v     = instance_v_ptr.Elem()
		field_t        reflect.StructField
		field_v        reflect.Value
		ok             bool
	)

	field_t, ok = instance_t.FieldByName(name)
	if !ok {
		panic(
			fmt.Sprintf("field %q not found in %T", name, instance),
		)
	}

	field_v = instance_v.FieldByIndex(field_t.Index)
	// field_v := instance_v.FieldByName(name)
	if !field_v.IsValid() {
		panic(
			fmt.Sprintf("field %q not found in %T", name, instance),
		)
	}

	return &FieldDef{
		Blank:          blank,
		Editable:       editable,
		instance_t_ptr: instance_t_ptr,
		instance_v_ptr: instance_v_ptr,
		instance_t:     instance_t,
		instance_v:     instance_v,
		field_t:        field_t,
		field_v:        field_v,
	}
}

func (f *FieldDef) IsBlank() bool {
	return f.Blank
}

func (f *FieldDef) IsEditable() bool {
	return f.Editable
}

func (f *FieldDef) GetValue() interface{} {
	return f.field_v.Interface()
}

func (f *FieldDef) SetValue(v interface{}, force bool) {
	var r_v = reflect.ValueOf(v)

	if !r_v.IsValid() {
		f.field_v.Set(reflect.Zero(f.field_t.Type))
		return
	}

	if r_v.Type() != f.field_t.Type && !r_v.CanConvert(f.field_t.Type) {
		panic(
			fmt.Sprintf("field %q (%q) is not compatible with type %T", f.field_t.Name, f.field_t.Type, v),
		)
	}

	if r_v.Type() != f.field_t.Type {
		r_v = r_v.Convert(f.field_t.Type)
	}

	if !f.field_v.CanSet() {
		panic(
			fmt.Sprintf("field %q is not settable", f.field_t.Name),
		)
	}

	if !f.Editable && !force {
		panic(
			fmt.Sprintf("field %q is not editable", f.field_t.Name),
		)
	}

	f.field_v.Set(r_v)
}
