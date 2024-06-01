package attrs

import (
	"encoding/json"
	"fmt"
	"net/mail"
	"reflect"
	"time"

	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/django/forms/widgets"
	"github.com/Nigel2392/goldcrest"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	HookFormFieldForType = "attrs.FormFieldForType"
)

var capCaser = cases.Title(language.English)

type FormFieldGetter func(f Field, t reflect.Type, v reflect.Value, opts ...func(fields.Field)) (fields.Field, bool)

type FieldDef struct {
	attrDef        FieldConfig
	instance_t_ptr reflect.Type
	instance_v_ptr reflect.Value
	instance_t     reflect.Type
	instance_v     reflect.Value
	field_t        reflect.StructField
	field_v        reflect.Value
	formField      fields.Field
}

func NewField[T any](instance *T, name string, conf *FieldConfig) *FieldDef {
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
	assert.True(ok, "field %q not found in %T", name, instance)

	field_v = instance_v.FieldByIndex(field_t.Index)
	assert.True(field_v.IsValid(), "field %q not found in %T", name, instance)

	if conf == nil {
		conf = &FieldConfig{}
	}

	return &FieldDef{
		attrDef:        *conf,
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
	if f.attrDef.Label != "" {
		return fields.T(f.attrDef.Label)
	}
	return fields.T(capCaser.String(f.field_t.Name))
}

func (f *FieldDef) HelpText() string {
	if helpTexter, ok := f.field_v.Interface().(Helper); ok {
		return fields.T(helpTexter.HelpText())
	}
	if f.attrDef.HelpText != "" {
		return fields.T(f.attrDef.HelpText)
	}
	return ""
}

func (f *FieldDef) Name() string {
	return f.field_t.Name
}

func (f *FieldDef) AllowNull() bool {
	return f.attrDef.Null
}

func (f *FieldDef) AllowBlank() bool {
	return f.attrDef.Blank
}

func (f *FieldDef) AllowEdit() bool {
	return !f.attrDef.ReadOnly
}

func (f *FieldDef) Validate() error {
	return nil
}

func (f *FieldDef) GetValue() interface{} {
	return f.field_v.Interface()
}

func (f *FieldDef) GetDefault() interface{} {

	var funcName = fmt.Sprintf("GetDefault%s", f.Name())
	if method, ok := f.instance_t.MethodByName(funcName); ok {
		return method.Func.Call([]reflect.Value{f.instance_v_ptr})[0].Interface()
	}

	if !f.field_v.IsValid() {
		return reflect.Zero(f.field_t.Type).Interface()
	}

	return f.field_v.Interface()
}

func (f *FieldDef) FormField() fields.Field {
	if f.formField != nil {
		return f.formField
	}

	var opts = make([]func(fields.Field), 0)

	opts = append(opts, fields.Label(f.Label))
	opts = append(opts, fields.HelpText(f.HelpText))

	var typForNew = f.field_t.Type
	if f.field_t.Type.Kind() == reflect.Ptr {
		typForNew = f.field_t.Type.Elem()
	}

	var formField fields.Field
	var hooks = goldcrest.Get[FormFieldGetter](HookFormFieldForType)
	for _, hook := range hooks {
		if field, ok := hook(f, typForNew, f.field_v, opts...); ok {
			formField = field
			goto returnField
		}
	}

	switch reflect.New(typForNew).Elem().Interface().(type) {
	case time.Time:
		formField = fields.DateField(widgets.DateWidgetTypeDateTime, opts...)
	case json.RawMessage:
		formField = fields.JSONField[map[string]interface{}](opts...)
	case mail.Address:
		formField = fields.EmailField(opts...)
	}

	if formField != nil {
		goto returnField
	}

	switch f.field_t.Type.Kind() {
	case reflect.String:
		formField = fields.CharField(opts...)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		formField = fields.NumberField[int](opts...)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		formField = fields.NumberField[uint](opts...)
	case reflect.Float32, reflect.Float64:
		formField = fields.NumberField[float64](opts...)
	default:
		formField = fields.CharField(opts...)
	}

returnField:
	formField.SetName(f.Name())
	f.formField = formField
	return formField
}

func (f *FieldDef) SetValue(v interface{}, force bool) error {
	var r_v = reflect.ValueOf(v)

	if err := assert.True(
		r_v.IsValid() || f.AllowNull(),
		"field %q (%q) is not valid", f.field_t.Name, f.field_t.Type,
	); err != nil {
		return err
	}

	if !r_v.IsValid() && f.AllowNull() {
		f.field_v.Set(reflect.Zero(f.field_t.Type))
		return nil
	}

	if err := assert.True(
		r_v.Type() == f.field_t.Type || r_v.CanConvert(f.field_t.Type),
		"field %q (%q) is not convertible to %q",
		f.field_t.Name, r_v.Type(), f.field_t.Type,
	); err != nil {
		return err
	}

	if r_v.Type() != f.field_t.Type {
		r_v = r_v.Convert(f.field_t.Type)
	}

	if err := assert.True(
		f.field_v.CanSet() && (f.AllowEdit() || force),
		"field %q is not editable", f.field_t.Name,
	); err != nil {
		return err
	}

	if r_v.IsZero() && !f.AllowBlank() {
		switch r_v.Kind() {
		case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
			reflect.Float32, reflect.Float64,
			reflect.Complex64, reflect.Complex128:
		default:
			return assert.Fail(
				fmt.Sprintf("field %q is not blank", f.field_t.Name),
			)
		}
	}

	f.field_v.Set(r_v)
	return nil
}
