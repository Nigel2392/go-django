package fattrs

import (
	"context"
	"database/sql/driver"
	"fmt"
	"reflect"

	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/elliotchance/orderedmap/v2"
)

var (
	_                        attrs.Field = (*ReflectlessField[attrs.Definer])(nil)
	DEFAULT_FIELD_CONFIG_KEY             = "reflectlessfield.config.key"
)

type FieldConfig[T attrs.Definer] struct {
	Config         attrs.FieldConfig
	GetValue       func(obj T) interface{}
	GetDriverValue func(obj T) (interface{}, error)
	SetValue       func(obj T, value any) error
	SetDriverValue func(obj T, value any) error
	Validate       func(obj T, value any) error
	Default        interface{}
}

func RegisterModel[T attrs.Definer](register func(addField func(string, FieldConfig[T]))) {
	var fieldMap = orderedmap.NewOrderedMap[string, FieldConfig[T]]()
	var addField = func(fieldName string, conf FieldConfig[T]) {
		fieldMap.Set(fieldName, conf)
	}

	register(addField)

	var rt = reflect.TypeFor[T]()
	for head := fieldMap.Front(); head != nil; head = head.Next() {
		AddOptions(rt, head.Key, fieldConfigToOptions(head.Key, head.Value))
	}
}

func fieldConfigToOptions[T attrs.Definer](name string, conf FieldConfig[T]) *reflectlessFieldOpts[T] {
	var opts = &reflectlessFieldOpts[T]{
		getValue:       conf.GetValue,
		getDriverValue: conf.GetDriverValue,
		setValue:       conf.SetValue,
		setDriverValue: conf.SetDriverValue,
		validateValue:  conf.Validate,
	}

	if opts.getValue == nil {
		panic("GetValue function must be provided")
	}

	if opts.setValue == nil {
		panic("SetValue function must be provided")
	}

	if opts.getDriverValue == nil {
		opts.getDriverValue = func(obj T) (interface{}, error) {
			return opts.getValue(obj), nil
		}
	}

	if opts.setDriverValue == nil {
		opts.setDriverValue = func(obj T, value any) error {
			return opts.setValue(obj, value)
		}
	}

	if opts.validateValue == nil {
		opts.validateValue = func(obj T, value any) error {
			return nil
		}
	}

	switch _default := conf.Default.(type) {
	case func(T) interface{}:
		opts.getDefault = _default
	case func(attrs.Definer) interface{}:
		opts.getDefault = func(obj T) interface{} {
			return _default(obj)
		}
	case NULL:
		opts.getDefault = func(obj T) interface{} {
			return nil
		}
	case nil:
		panic("default value must be provided")
	default:
		opts.getDefault = func(obj T) interface{} {
			return _default
		}
	}

	opts.FieldDefinition = attrs.NewField(
		attrs.NewObject[T](context.Background(), reflect.TypeFor[T]()),
		name, &conf.Config,
	)

	return opts
}

type reflectlessFieldOpts[T attrs.Definer] struct {
	attrs.FieldDefinition
	getValue       func(obj T) interface{}
	getDriverValue func(obj T) (interface{}, error)
	setValue       func(obj T, value any) error
	setDriverValue func(obj T, value any) error
	getDefault     func(obj T) interface{}
	validateValue  func(obj T, value any) error
}

type ReflectlessField[T attrs.Definer] struct {
	obj       T
	defs      attrs.Definitions
	fieldName string
	opts      *reflectlessFieldOpts[T]
	confFunc  func() FieldConfig[T]
}

type NULL struct{}

func NewField[T attrs.Definer](obj T, name string, cnf func() FieldConfig[T]) attrs.Field {
	var opts, ok = GetOptions[*reflectlessFieldOpts[T]](reflect.TypeOf(obj), name)
	if !ok && cnf == nil {
		panic(fmt.Sprintf("options were not configured for field %q in type %T", name, obj))
	}

	var f = &ReflectlessField[T]{
		obj:       obj,
		fieldName: name,
		opts:      opts,
	}

	if !ok {
		f.confFunc = cnf
	}

	return f
}

func (r *ReflectlessField[T]) useOpts() *reflectlessFieldOpts[T] {
	if r.opts != nil {
		return r.opts
	}

	if r.confFunc != nil {
		r.opts = fieldConfigToOptions(r.fieldName, r.confFunc())

		AddOptions(
			reflect.TypeOf(r.obj),
			r.fieldName,
			r.opts,
		)
	}

	return r.opts
}

func (r *ReflectlessField[T]) OnModelRegister(model attrs.Definer, outer attrs.FieldDefinition) error {
	if r.opts == nil && r.confFunc != nil {
		AddOptions(
			reflect.TypeOf(r.obj),
			r.fieldName,
			fieldConfigToOptions(r.fieldName, r.confFunc()),
		)
	}
	return nil
}

func (r *ReflectlessField[T]) StructField() *reflect.StructField {
	var rv = reflect.TypeOf(r.obj)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}

	var fld, ok = rv.FieldByName(r.fieldName)
	if !ok {
		return nil
	}

	return &fld
}

func (r *ReflectlessField[T]) Name() string                                    { return r.fieldName }
func (r *ReflectlessField[T]) FieldDefinitions() attrs.Definitions             { return r.defs }
func (r *ReflectlessField[T]) BindToDefinitions(definitions attrs.Definitions) { r.defs = definitions }
func (r *ReflectlessField[T]) HelpText(ctx context.Context) string             { return r.useOpts().HelpText(ctx) }
func (r *ReflectlessField[T]) Label(ctx context.Context) string                { return r.useOpts().Label(ctx) }
func (r *ReflectlessField[T]) Instance() attrs.Definer                         { return r.obj }
func (r *ReflectlessField[T]) ColumnName() string                              { return r.useOpts().ColumnName() }
func (r *ReflectlessField[T]) Type() reflect.Type                              { return r.useOpts().Type() }
func (r *ReflectlessField[T]) Attrs() map[string]any                           { return r.useOpts().Attrs() }
func (r *ReflectlessField[T]) Rel() attrs.Relation                             { return r.useOpts().Rel() }
func (r *ReflectlessField[T]) IsPrimary() bool                                 { return r.useOpts().IsPrimary() }
func (r *ReflectlessField[T]) AllowNull() bool                                 { return r.useOpts().AllowNull() }
func (r *ReflectlessField[T]) AllowBlank() bool                                { return r.useOpts().AllowBlank() }
func (r *ReflectlessField[T]) AllowEdit() bool                                 { return r.useOpts().AllowEdit() }
func (r *ReflectlessField[T]) FormField() fields.Field                         { return r.useOpts().FormField() }
func (r *ReflectlessField[T]) ToString() string                                { return attrs.ToString(r.GetValue()) }

func (r *ReflectlessField[T]) Scan(v any) error {
	return r.useOpts().setDriverValue(r.obj, v)
}

func (r *ReflectlessField[T]) Value() (driver.Value, error) {
	return r.useOpts().getDriverValue(r.obj)
}

func (r *ReflectlessField[T]) GetValue() interface{} {
	return r.useOpts().getValue(r.obj)
}

func (r *ReflectlessField[T]) GetDefault() interface{} {
	return r.useOpts().getDefault(r.obj)
}

func (r *ReflectlessField[T]) SetValue(v interface{}, _ bool) error {
	return r.useOpts().setValue(r.obj, v)
}

func (r *ReflectlessField[T]) Validate() error {
	return r.useOpts().validateValue(r.obj, r.GetValue())
}
