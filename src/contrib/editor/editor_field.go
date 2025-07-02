package editor

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	_ "unsafe"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/attrutils"
	"github.com/Nigel2392/go-django/src/forms/fields"
)

var (
	_ attrs.Field = (*Field)(nil)
)

type FieldConfig struct {
	// Label is the label for the field.
	Label string

	// HelpText is the help text for the field.
	HelpText string

	// Features is a list of features to enable for the editor.
	Features []string

	// Default is the default value for the field.
	Default *EditorJSBlockData

	// ReadOnly indicates if the field is read-only.
	ReadOnly bool

	// Blank indicates if the field can be blank.
	Blank bool

	// Nullable indicates if the field can be null.
	Nullable bool

	// Column is the name of the column in the database.
	Column string

	// Attributes are additional attributes for the field.
	Attributes map[string]any

	// Options for the form field.
	FormFieldOpts []func(fields.Field)
}

type _etter[T any] struct {
	pointer T
	direct  T
}

type Field struct {
	name     string
	config   FieldConfig
	fieldT   reflect.StructField
	fieldV   reflect.Value
	instance attrs.Definer
	defs     attrs.Definitions
	setter   _etter[func(f *Field, v reflect.Value) error]
}

var (
	_BLOCK_DATA_TYPE_PTR = reflect.TypeOf((*EditorJSBlockData)(nil))
	_BLOCK_DATA_TYPE     = _BLOCK_DATA_TYPE_PTR.Elem()
)

func NewField(forModel attrs.Definer, name string, cnf ...FieldConfig) *Field {
	if forModel == nil {
		panic("NewField: model is nil")
	}

	if name == "" {
		panic("NewField: name is empty")
	}

	var conf FieldConfig
	if len(cnf) > 0 {
		conf = cnf[0]
	}

	var (
		modelT = reflect.TypeOf(forModel)
		modelV = reflect.ValueOf(forModel)
	)

	if modelT.Kind() != reflect.Pointer || modelT.Elem().Kind() != reflect.Struct {
		panic(fmt.Errorf("NewField: model %T is not a pointer to a struct", forModel))
	}

	modelT = modelT.Elem()
	modelV = modelV.Elem()

	var structField, ok = attrutils.GetStructField(modelT, name)
	if !ok {
		panic(fmt.Errorf("NewField: model %T has no field %q", forModel, name))
	}

	var structFieldType = structField.Type
	if structFieldType.Kind() == reflect.Pointer {
		// If the field is a pointer, we need to get the element type
		structFieldType = structFieldType.Elem()
	}

	if structFieldType != _BLOCK_DATA_TYPE {
		panic(fmt.Errorf("NewField: field %T.%s is not of type EditorJSBlockData", forModel, name))
	}

	// List of setters and getters to be used
	// for scanning and setting values in the field.
	// This allows for simpler handling of different types
	// of data models and their fields.
	var (
		fieldV  = modelV.FieldByIndex(structField.Index)
		setters = _etter[func(f *Field, v reflect.Value) error]{}
	)

	switch {
	case structFieldType.Kind() == reflect.Pointer:
		setters.pointer = func(f *Field, v reflect.Value) error {
			// is ok to straight up set the value
			if v.IsNil() {
				fieldV.Set(reflect.Zero(structFieldType))
				return nil
			}
			fieldV.Set(v)
			return nil
		}
		setters.direct = func(f *Field, v reflect.Value) error {
			if !v.IsValid() || v.Kind() == reflect.Pointer && v.IsNil() {
				fieldV.Set(reflect.Zero(structFieldType))
				return nil
			}
			var vPtr = reflect.New(structFieldType.Elem())
			if v.Kind() == reflect.Pointer {
				vPtr.Elem().Set(v.Elem())
			} else {
				vPtr.Elem().Set(v)
			}
			fieldV.Set(vPtr)
			return nil
		}
	default:
		setters.pointer = func(f *Field, v reflect.Value) error {
			// is ok to straight up set the value
			if v.IsNil() {
				fieldV.Set(reflect.Zero(structFieldType))
				return nil
			}
			fieldV.Set(v.Elem())
			return nil
		}
		setters.direct = func(f *Field, v reflect.Value) error {
			if !v.IsValid() || v.Kind() == reflect.Pointer && v.IsNil() {
				fieldV.Set(reflect.Zero(structFieldType))
				return nil
			}
			fieldV.Set(v)
			return nil
		}
	}

	var f = &Field{
		name:     name,
		config:   conf,
		fieldT:   structField,
		fieldV:   fieldV,
		instance: forModel,
		setter:   setters,
	}

	return f
}

func (f *Field) FieldDefinitions() attrs.Definitions {
	return f.defs
}

func (f *Field) BindToDefinitions(defs attrs.Definitions) {
	f.defs = defs
}

func (f *Field) setupInitialVal() {

	assert.Err(attrs.BindValueToModel(
		f.instance, f, f.fieldV,
	))

	var (
		data *EditorJSBlockData
		ok   bool
	)
	switch {
	case f.fieldV.Kind() == reflect.Pointer && !f.fieldV.IsNil():
		data, ok = f.fieldV.Interface().(*EditorJSBlockData)
	case f.fieldV.Kind() == reflect.Struct:
		data, ok = f.fieldV.Addr().Interface().(*EditorJSBlockData)
	}
	if !ok {
		panic(fmt.Errorf("Field %s in model %T is not of type *EditorJSBlockData", f.name, f.instance))
	}

	if data != nil {
		data.Features = Features(f.config.Features...)
	}

}

func (f *Field) Name() string {
	return f.name
}

// no real column, special case for virtual fields
func (e *Field) ColumnName() string {
	if e.config.Column != "" {
		return e.config.Column
	}
	return attrs.ColumnName(e.Name())
}

func (e *Field) Tag(s string) string {
	return e.fieldT.Tag.Get(s)
}

func (e *Field) Type() reflect.Type {
	return _BLOCK_DATA_TYPE_PTR
}

func (e *Field) Attrs() map[string]any {
	var m = e.config.Attributes
	if m == nil {
		m = make(map[string]interface{})
	}
	m[attrs.AttrNameKey] = e.Name()
	m[attrs.AttrAllowNullKey] = e.AllowNull()
	m[attrs.AttrAllowBlankKey] = e.AllowBlank()
	m[attrs.AttrAllowEditKey] = e.AllowEdit()
	m[attrs.AttrIsPrimaryKey] = e.IsPrimary()
	return m
}

func (e *Field) IsPrimary() bool {
	return false
}

func (e *Field) AllowNull() bool {
	return e.config.Nullable
}

func (e *Field) AllowBlank() bool {
	return e.config.Blank
}

func (e *Field) AllowEdit() bool {
	return !e.config.ReadOnly
}

func (e *Field) AllowDBEdit() bool {
	return true
}

func (e *Field) GetValue() interface{} {
	var val interface{}
	switch {
	case !e.fieldV.IsValid() || (e.fieldV.Kind() == reflect.Pointer && e.fieldV.IsNil()):
		return nil
	case e.fieldV.Kind() == reflect.Pointer:
		val = e.fieldV.Interface()
	default:
		// is ok to return the addr - fieldV is a structfield
		val = e.fieldV.Addr().Interface()
	}

	assert.Err(attrs.BindValueToModel(
		e.instance, e, val,
	))

	return val
}

func (e *Field) SetValue(v interface{}, force bool) error {
	var (
		rV = reflect.ValueOf(v)
		rT = reflect.TypeOf(v)
	)

	if !rV.IsValid() || rT == nil {
		e.fieldV.Set(reflect.Zero(e.fieldT.Type))
		return nil
	}

	defer e.setupInitialVal()

	switch {
	case rV.Kind() == reflect.String:
		var data, err = JSONUnmarshalEditorData(e.config.Features, []byte(rV.String()))
		if err != nil {
			return fmt.Errorf("failed to unmarshal EditorJSBlockData from string: %w", err)
		}

		rT = reflect.TypeOf(data)
		rV = reflect.ValueOf(data)
	case rV.Kind() == reflect.Slice && rT.Elem().Kind() == reflect.Uint8:
		var data, err = JSONUnmarshalEditorData(e.config.Features, rV.Bytes())
		if err != nil {
			return fmt.Errorf("failed to unmarshal EditorJSBlockData from byte slice: %w", err)
		}

		rT = reflect.TypeOf(data)
		rV = reflect.ValueOf(data)
	}

	if !rV.IsValid() || rT == nil {
		rT = rV.Type()
	}

	if !force {
		e.defs.SignalChange(e, v)
	}

	switch rT.Kind() {
	case reflect.Ptr:
		if err := e.setter.pointer(e, rV); err != nil {
			return fmt.Errorf("failed to set value for field %s: %w", e.Name(), err)
		}
		return nil

	case reflect.Struct:
		if err := e.setter.direct(e, rV); err != nil {
			return fmt.Errorf("failed to set value for field %s: %w", e.Name(), err)
		}
		return nil
	}

	return fmt.Errorf("cannot set value for field %s: expected pointer or struct, got %s", e.Name(), rT.Kind())
}

func (e *Field) Value() (driver.Value, error) {
	var val = e.GetValue()
	if val == nil {
		return nil, nil
	}

	var obj, ok = val.(*EditorJSBlockData)
	if !ok {
		return nil, fmt.Errorf("value %v (%T) is not of type *EditorJSBlockData", val, val)
	}

	if obj == nil {
		return nil, nil
	}

	var bytes, err = JSONMarshalEditorData(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal EditorJSBlockData: %w", err)
	}

	return string(bytes), nil
}

func (e *Field) Scan(src interface{}) error {
	return e.SetValue(src, true)
}

func (e *Field) GetDefault() interface{} {
	return e.config.Default
}

func (e *Field) Instance() attrs.Definer {
	return e.instance
}

func (e *Field) Rel() attrs.Relation {
	return nil
}

func (e *Field) FormField() fields.Field {
	var opts = make([]func(fields.Field), 0, len(e.config.FormFieldOpts)+1)
	opts = append(opts, func(f fields.Field) {
		f.SetName(e.Name())
		f.SetLabel(e.Label)
		f.SetHelpText(e.HelpText)

		if e.config.ReadOnly {
			f.SetReadOnly(true)
		}

		if !e.config.Blank {
			f.SetRequired(true)
		}
	})

	return EditorJSField(e.config.Features, append(opts, e.config.FormFieldOpts...)...)
}

func (e *Field) Validate() error {
	return nil
}

func (e *Field) Label() string {
	if e.config.Label != "" {
		return e.config.Label
	}
	return e.Name()
}

func (e *Field) ToString() string {
	return fmt.Sprint(e.GetValue())
}

func (e *Field) HelpText() string {
	return e.config.HelpText
}
