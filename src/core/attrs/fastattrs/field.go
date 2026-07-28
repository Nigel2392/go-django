package fastattrs

import (
	"context"
	"database/sql/driver"
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
		AddOptions[T](rt, head.Key, fieldConfigToOptions(head.Key, head.Value))
	}
}

func fieldConfigToOptions[T attrs.Definer](name string, conf FieldConfig[T]) *reflectlessFieldOpts[T] {
	var opts = &reflectlessFieldOpts[T]{
		getValue:       conf.GetValue,
		getDriverValue: conf.GetDriverValue,
		setValue:       conf.SetValue,
		setDriverValue: conf.SetDriverValue,
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
}

type ReflectlessField[T attrs.Definer] struct {
	obj       T
	defs      attrs.Definitions
	fieldName string
	opts      *reflectlessFieldOpts[T]
}

type NULL struct{}

func NewField[T attrs.Definer](obj T, name string) attrs.Field {
	var (
		rtyp = reflect.TypeOf(obj)
		opts = GetOptions[T](rtyp, name)
	)

	return &ReflectlessField[T]{
		obj:       obj,
		fieldName: name,
		opts:      opts,
	}
}

func (r *ReflectlessField[T]) Name() string                                    { return r.fieldName }
func (r *ReflectlessField[T]) FieldDefinitions() attrs.Definitions             { return r.defs }
func (r *ReflectlessField[T]) BindToDefinitions(definitions attrs.Definitions) { r.defs = definitions }
func (r *ReflectlessField[T]) HelpText(ctx context.Context) string             { return r.opts.HelpText(ctx) }
func (r *ReflectlessField[T]) Label(ctx context.Context) string                { return r.opts.Label(ctx) }
func (r *ReflectlessField[T]) Instance() attrs.Definer                         { return r.obj }
func (r *ReflectlessField[T]) ColumnName() string                              { return r.opts.ColumnName() }
func (r *ReflectlessField[T]) Type() reflect.Type                              { return r.opts.Type() }
func (r *ReflectlessField[T]) Attrs() map[string]any                           { return r.opts.Attrs() }
func (r *ReflectlessField[T]) Rel() attrs.Relation                             { return r.opts.Rel() }
func (r *ReflectlessField[T]) IsPrimary() bool                                 { return r.opts.IsPrimary() }
func (r *ReflectlessField[T]) AllowNull() bool                                 { return r.opts.AllowNull() }
func (r *ReflectlessField[T]) AllowBlank() bool                                { return r.opts.AllowBlank() }
func (r *ReflectlessField[T]) AllowEdit() bool                                 { return r.opts.AllowEdit() }
func (r *ReflectlessField[T]) FormField() fields.Field                         { return r.opts.FormField() }
func (r *ReflectlessField[T]) Scan(v any) error                                { return r.opts.setDriverValue(r.obj, v) }
func (r *ReflectlessField[T]) Value() (driver.Value, error)                    { return r.opts.getDriverValue(r.obj) }
func (r *ReflectlessField[T]) ToString() string                                { return attrs.ToString(r.GetValue()) }
func (r *ReflectlessField[T]) GetValue() interface{}                           { return r.opts.getValue(r.obj) }
func (r *ReflectlessField[T]) GetDefault() interface{}                         { return r.opts.getDefault(r.obj) }
func (r *ReflectlessField[T]) SetValue(v interface{}, _ bool) error            { return r.opts.setValue(r.obj, v) }
func (r *ReflectlessField[T]) Validate() error                                 { return nil }
