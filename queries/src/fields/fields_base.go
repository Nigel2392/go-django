package fields

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/forms/fields"
)

var _ attrs.Field = &DataModelField[any]{}

type DataModelField[T any] struct {
	// model is the model that this field belongs to
	Model attrs.Definer

	// defs is the definitions of the model
	defs attrs.Definitions

	// dataModel is the model that contains the data for this field
	//
	// it should be embedded in the attrs.Definer type which this virtual field is for
	dataModel any

	// val is the value of the field in the model
	val reflect.Value

	// name is the name of the field's map key in the model
	// it is also the alias used in the query
	name string

	// _Type is the type of the result of the expression
	_Type reflect.Type

	// fieldRef is the back reference in case this field is embedded in another
	// field type
	fieldRef attrs.Field

	setters []func(f *DataModelField[T], v any) error
	getters []func(f *DataModelField[T]) (any, bool)
}

func typeName(t reflect.Type) string {
	var d int
	d, t = indirect(t)
	return fmt.Sprintf("%s%s", strings.Repeat("*", d), t.Name())
}

func indirect(t reflect.Type) (int, reflect.Type) {
	return _indirect(0, t)
}

func _indirect(depth int, t reflect.Type) (int, reflect.Type) {
	if t.Kind() == reflect.Ptr {
		return _indirect(depth+1, t.Elem())
	}
	return depth, t
}

var (
	_DATA_MODEL_TYPE       = reflect.TypeOf((*queries.DataModel)(nil)).Elem()
	_DATA_MODEL_STORE_TYPE = reflect.TypeOf((*queries.ModelDataStore)(nil)).Elem()
	_DEFINER_TYPE          = reflect.TypeOf((*attrs.Definer)(nil)).Elem()
)

func scannerGetter[T any](f *DataModelField[T]) (any, bool) {
	if !f.val.IsValid() || !f.val.CanInterface() {
		return nil, false
	}
	var val = f.val.Interface()
	if val == nil {
		return nil, false
	}
	return val.(T), true
}

func scannerSetter[T any](f *DataModelField[T], v any) error {
	if v == nil {
		f.val.Set(reflect.Zero(f._Type))
		return nil
	}

	var vVal = reflect.ValueOf(v)
	if !vVal.Type().AssignableTo(f._Type) {
		panic(fmt.Errorf(
			"NewDataModelField: cannot assign %T to %s (%s != %s)",
			v, typeName(f._Type), typeName(vVal.Type()), typeName(f._Type),
		))
	}

	if !f.val.IsValid() {
		return fmt.Errorf("NewDataModelField: dst value is nil")
	}
	if !f.val.CanSet() {
		return fmt.Errorf("NewDataModelField: dst value is not settable")
	}

	f.val.Set(vVal)
	return nil
}

func dataModelGetter[T any](f *DataModelField[T]) (any, bool) {
	switch m := f.dataModel.(type) {
	case queries.ModelDataStore:
		return m.GetValue(f.name)
	case queries.DataModel:
		return m.DataStore().GetValue(f.name)
	}
	return nil, false
}

func dataModelsetter[T any](f *DataModelField[T], v any) error {
	switch m := f.dataModel.(type) {
	case queries.ModelDataStore:
		return m.SetValue(f.name, v)
	case queries.DataModel:
		return m.DataStore().SetValue(f.name, v)
	}
	return nil
}

type DataModelFieldConfig struct {
	ResultType reflect.Type // Type of the result of the expression
	Ref        attrs.Field  // Reference to the field in the model
}

func NewDataModelField[T any](forModel attrs.Definer, dst any, name string, cnf ...DataModelFieldConfig) *DataModelField[T] {
	if forModel == nil || dst == nil {
		panic("NewDataModelField: model is nil")
	}

	if name == "" {
		panic("NewDataModelField: name is empty")
	}

	var conf DataModelFieldConfig
	if len(cnf) > 0 {
		conf = cnf[0]
	}

	var (
		Type             = reflect.TypeOf((*T)(nil)).Elem()
		originalT        = reflect.TypeOf(dst)
		originalV        = reflect.ValueOf(dst)
		dstT             = originalT
		dstV             = originalV
		isPointer        = dstT.Kind() == reflect.Pointer
		isPointerPointer = dstT.Kind() == reflect.Pointer &&
			dstT.Elem().Kind() == reflect.Pointer
	)

	if conf.ResultType != nil {
		Type = conf.ResultType
	}

	// List of setters and getters to be used
	// for scanning and setting values in the field.
	// This allows for simpler handling of different types
	// of data models and their fields.
	var (
		setters = make([]func(f *DataModelField[T], v any) error, 0, 2)
		getters = make([]func(f *DataModelField[T]) (any, bool), 0, 2)
	)
	switch {
	case dstT.Kind() == reflect.Pointer && dstT.Elem() == Type:
		// Scan the value into a pointer to T

		dstT = dstT.Elem()
		dstV = dstV.Elem()

		if !dstV.IsValid() || dstV.Kind() == reflect.Pointer && dstV.IsNil() {
			if isPointerPointer {
				var newVal = reflect.New(dstT.Elem())
				dstV.Set(newVal)
			} else {
				var newVal = reflect.New(dstT)
				dstV.Set(newVal)
			}
		}

		getters = append(getters, scannerGetter[T])
		setters = append(setters, scannerSetter[T])

	case isPointer && dstT.Elem().Kind() == reflect.Struct:
		// Scan the value into a *struct field

		if !dstV.IsValid() || dstV.IsNil() {
			panic(fmt.Errorf("NewDataModelField: dst value is nil for %T.%s", forModel, name))
		}

		dstT = dstT.Elem()
		dstV = dstV.Elem()
		var field, ok = dstT.FieldByName(name)
		if !ok || !field.IsExported() {
			break
		}

		if !field.Type.AssignableTo(Type) && !field.Type.ConvertibleTo(Type) {
			panic(fmt.Errorf("NewDataModelField: %s != %s (%T.%s)", typeName(Type), typeName(field.Type), forModel, name))
		}

		var fieldVal = dstV.FieldByIndex(field.Index)
		if !fieldVal.IsValid() {
			if !fieldVal.CanSet() {
				panic(fmt.Errorf("NewDataModelField: field %T.%s is not settable", forModel, name))
			}

			// if the field is not valid, we need to create a new value for it
			if field.Type.Kind() == reflect.Ptr {
				fieldVal.Set(reflect.New(field.Type.Elem()))
			} else {
				fieldVal.Set(reflect.Zero(field.Type))
			}
		}

		dstV = fieldVal

		getters = append(getters, scannerGetter[T])
		setters = append(setters, scannerSetter[T])

	default:
		panic(fmt.Errorf(
			"NewDataModelField: _Type %s is not a pointer to a struct, a pointer to T or implements queries.DataModel or queries.ModelDataStore",
			typeName(dstT),
		))
	}

	// Always check if the original type
	// implements DataModel or ModelDataStore
	// to allow for custom data models that are
	// not pointers to structs/values.
	var dataModel any
	if originalT.Implements(_DATA_MODEL_TYPE) || originalT.Implements(_DATA_MODEL_STORE_TYPE) {
		dataModel = originalV.Interface()
		getters = append(getters, dataModelGetter[T])
		setters = append(setters, dataModelsetter[T])
	}

	var f = &DataModelField[T]{
		Model:     forModel,
		dataModel: dataModel,
		val:       dstV,
		_Type:     Type,
		name:      name,
		fieldRef:  conf.Ref,
		getters:   getters,
		setters:   setters,
	}

	return f
}

func (f *DataModelField[T]) signalChange(val any) {
	if f.fieldRef != nil {
		f.defs.SignalChange(f.fieldRef, val)
	} else {
		f.defs.SignalChange(f, val)
	}
}

func (f *DataModelField[T]) FieldDefinitions() attrs.Definitions {
	return f.defs
}

func (f *DataModelField[T]) BindToDefinitions(defs attrs.Definitions) {
	f.defs = defs
}

func (f *DataModelField[T]) setupInitialVal() {
	if f.val.IsValid() && (f.val.Kind() == reflect.Pointer && !f.val.IsNil()) {
		f.bindVal(f.val)
	}
}

func (f *DataModelField[T]) bindVal(val any) {
	switch {
	case f.fieldRef != nil:
		assert.Err(attrs.BindValueToModel(
			f.Model, f.fieldRef, val,
		))
	default:
		assert.Err(attrs.BindValueToModel(
			f.Model, f, val,
		))
	}
}

func (f *DataModelField[T]) getQueryValue() (any, bool) {
	if len(f.getters) == 0 {
		panic(fmt.Errorf("getQueryValue: no getters defined for %s", f.name))
	}

	var val any
	var ok bool
	for _, getter := range f.getters {
		val, ok = getter(f)
		if ok {
			return val, true
		}
	}

	return nil, false
}

func (f *DataModelField[T]) setQueryValue(v any) (err error) {
	// f.signalChange(v)

	for _, setter := range f.setters {
		if err = setter(f, v); err != nil {
			panic(fmt.Errorf("setQueryValue: cannot set value for %s: %w", f.name, err))
		}
	}

	return nil
}

func (f *DataModelField[T]) Name() string {
	return f.name
}

// no real column, special case for virtual fields
func (e *DataModelField[T]) ColumnName() string {
	return ""
}

func (e *DataModelField[T]) Tag(string) string {
	return ""
}

func (e *DataModelField[T]) Type() reflect.Type {
	if e._Type == nil {
		panic("_Type is nil")
	}

	return e._Type
}

func (e *DataModelField[T]) Attrs() map[string]any {
	return map[string]any{}
}

func (e *DataModelField[T]) IsPrimary() bool {
	return false
}

func (e *DataModelField[T]) AllowNull() bool {
	return true
}

func (e *DataModelField[T]) AllowBlank() bool {
	return true
}

func (e *DataModelField[T]) AllowEdit() bool {
	return false
}

func (e *DataModelField[T]) AllowDBEdit() bool {
	return false
}

func (e *DataModelField[T]) GetValue() interface{} {
	var val, _ = e.getQueryValue()
	if e._Type.Kind() == reflect.Pointer && (e._Type.Comparable() && any(val) == any(*new(T)) || val == nil) {
		val = reflect.New(e._Type.Elem()).Interface()
		assert.Err(e.setQueryValue(val))
	}

	if val != nil {
		e.bindVal(val)
	}

	valTyped, ok := val.(T)
	if !ok {
		return *new(T)
	}

	return valTyped
}

func castToNumber[T any](s string) (any, error) {
	var n, err = attrs.CastToNumber[T](s)
	return n, err
}

var reflect_convert = map[reflect.Kind]func(string) (any, error){
	reflect.Int:     castToNumber[int],
	reflect.Int8:    castToNumber[int8],
	reflect.Int16:   castToNumber[int16],
	reflect.Int32:   castToNumber[int32],
	reflect.Int64:   castToNumber[int64],
	reflect.Uint:    castToNumber[uint],
	reflect.Uint8:   castToNumber[uint8],
	reflect.Uint16:  castToNumber[uint16],
	reflect.Uint32:  castToNumber[uint32],
	reflect.Uint64:  castToNumber[uint64],
	reflect.Float32: castToNumber[float32],
	reflect.Float64: castToNumber[float64],
	reflect.String: func(s string) (any, error) {
		return s, nil
	},
	reflect.Bool: func(s string) (any, error) {
		var b, err = strconv.ParseBool(s)
		return b, err
	},
}

var baseReflectKinds = (func() []reflect.Kind {
	var kinds = make([]reflect.Kind, 0, len(reflect_convert))
	for k := range reflect_convert {
		kinds = append(kinds, k)
	}
	return kinds
})()

func (e *DataModelField[T]) SetValue(v interface{}, force bool) error {
	var (
		rV = reflect.ValueOf(v)
		rT = reflect.TypeOf(v)
	)

	if !rV.IsValid() || rT == nil {
		rV = reflect.New(e._Type).Elem()
		rT = rV.Type()
	}

	e.bindVal(rV)

	if rT != e._Type {

		// check if the value implements the Definer interface
		// if it is the definer interface itself, the value cannot be created, we skip this.
		if e._Type.Implements(_DEFINER_TYPE) && e._Type != _DEFINER_TYPE {
			var obj = e.GetValue()
			if obj == nil {
				obj = attrs.NewObject[attrs.Definer](e._Type)
				if err := e.SetValue(obj, false); err != nil {
					return fmt.Errorf("cannot set value %v to %T: %w", v, *new(T), err)
				}
			}
			var defObj = obj.(attrs.Definer)
			var defs = defObj.FieldDefs()
			var prim = defs.Primary()
			return prim.Scan(v)
		}

		if rT.ConvertibleTo(e._Type) {
			rV = rV.Convert(e._Type)
		} else if rV.IsValid() && rT.Kind() == reflect.Ptr && (rT.Elem() == e._Type || rT.Elem().ConvertibleTo(e._Type)) {
			rV = rV.Elem()
			if rT.Elem() != e._Type {
				rV = rV.Convert(e._Type)
			}
		}

		if slices.Contains(baseReflectKinds, rT.Kind()) {

			if f, ok := reflect_convert[e._Type.Kind()]; ok {
				var val, err = f(rV.String())
				if err != nil {
					return fmt.Errorf("cannot convert %v to %T: %w", v, *new(T), err)
				}

				rV = reflect.ValueOf(val)

				if rV.Type() != e._Type {
					rV = rV.Convert(e._Type)
				}
			} else {
				return fmt.Errorf("cannot convert %v to %T", v, *new(T))
			}

		}
	}

	v = rV.Interface()

	if v == nil {
		reflect.New(e._Type).Interface()
	}

	if !force {
		e.signalChange(v)
	}

	if _, ok := v.(T); ok {
		e.setQueryValue(v)
		return nil
	}

	return fmt.Errorf("value %v (%T) is not of type %s", v, v, typeName(e._Type))
}

func (e *DataModelField[T]) Value() (driver.Value, error) {
	var val = e.GetValue()
	if val == nil {
		return *new(T), nil
	}

	switch v := val.(type) {
	case attrs.Definer:
		var pk = v.FieldDefs().Primary()
		if pk == nil {
			return nil, fmt.Errorf("Value: model %T has no primary key", v)
		}
		return pk.Value()
	case driver.Valuer:
		return v.Value()
	}

	return val, nil
}

func (e *DataModelField[T]) Scan(src interface{}) error {
	return e.SetValue(src, true)
}

func (e *DataModelField[T]) GetDefault() interface{} {
	return nil
}

func (e *DataModelField[T]) Instance() attrs.Definer {
	if e.Model == nil {
		panic("model is nil")
	}
	return e.Model
}

func (e *DataModelField[T]) Rel() attrs.Relation {
	return nil
}

func (e *DataModelField[T]) FormField() fields.Field {
	return nil
}

func (e *DataModelField[T]) Validate() error {
	return nil
}

func (e *DataModelField[T]) Label() string {
	return e.name
}

func (e *DataModelField[T]) ToString() string {
	return fmt.Sprint(e.GetValue())
}

func (e *DataModelField[T]) HelpText() string {
	return ""
}
