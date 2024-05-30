package attrs

import (
	"fmt"
	"reflect"
)

type FieldDef struct {
	Blank          bool
	Null           bool
	Editable       bool
	instance_t_ptr reflect.Type
	instance_v_ptr reflect.Value
	instance_t     reflect.Type
	instance_v     reflect.Value
	field_t        reflect.StructField
	field_v        reflect.Value
}

func NewField[T any](instance *T, name string, null, blank, editable bool) *FieldDef {
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
		Null:           null,
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

func (f *FieldDef) Label() string {
	if labeler, ok := f.field_v.Interface().(Labeler); ok {
		return labeler.Label()
	}
	return ""
}

func (f *FieldDef) HelpText() string {
	if helpTexter, ok := f.field_v.Interface().(Helper); ok {
		return helpTexter.HelpText()
	}
	return ""
}

func (f *FieldDef) Name() string {
	return f.field_t.Name
}

func (f *FieldDef) AllowNull() bool {
	return f.Null
}

func (f *FieldDef) AllowBlank() bool {
	return f.Blank
}

func (f *FieldDef) AllowEdit() bool {
	return f.Editable
}

func (f *FieldDef) GetValue() interface{} {
	return f.field_v.Interface()
}

func (f *FieldDef) SetValue(v interface{}, force bool) {
	var r_v = reflect.ValueOf(v)

	if !r_v.IsValid() && f.AllowNull() {
		f.field_v.Set(reflect.Zero(f.field_t.Type))
		return
	} else if !r_v.IsValid() {
		panic(
			fmt.Sprintf("field %q (%q) is not valid", f.field_t.Name, f.field_t.Type),
		)
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

	if r_v.IsZero() && (!f.AllowBlank() || !f.AllowNull()) {
		panic(
			fmt.Sprintf("field %q is not blank", f.field_t.Name),
		)
	}

	f.field_v.Set(r_v)
}
