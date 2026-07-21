package attrs

import (
	"context"
	"database/sql/driver"
	"fmt"
	"reflect"

	"github.com/Nigel2392/go-django/internal/django_reflect"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/assert"
)

func fieldNames(d any, exclude []string) []string {
	var excludeMap = make(map[string]struct{})
	for _, name := range exclude {
		excludeMap[name] = struct{}{}
	}

	var fields []FieldDefinition
	switch d := d.(type) {
	case Definer:
		var meta = GetModelMeta(d)
		var defs = meta.Definitions()
		fields = defs.Fields()
	case Definitions:
		var f = d.Fields()
		fields = make([]FieldDefinition, len(f))
		for i, field := range f {
			fields[i] = field
		}
	case StaticDefinitions:
		fields = d.Fields()
	case []Field:
		fields = make([]FieldDefinition, len(d))
		for i, f := range d {
			fields[i] = f
		}
	case []FieldDefinition:
		fields = d

	case []string: // []string is the only case which itself will return

		if len(excludeMap) == 0 {
			return d
		}

		var ret = make([]string, 0, len(d))
		for _, name := range d {
			if _, ok := excludeMap[name]; ok {
				continue
			}

			ret = append(ret, name)
		}
		return ret

	default:
		panic(fmt.Sprintf(
			"fieldNames: expected Definer, []Field or []FieldDefinition, got %T",
			d,
		))
	}

	var names = make([]string, 0, len(fields))
	for _, f := range fields {
		if _, ok := excludeMap[f.Name()]; ok {
			continue
		}
		names = append(names, f.Name())
	}

	return names
}

// A shortcut for getting the names of all fields in a Definer.
//
// The exclude parameter can be used to exclude certain fields from the result.
//
// This function is useful when you need to get the names of all fields in a
// model, but you want to exclude certain fields (e.g. fields that are not editable).
func FieldNames(d any, exclude []string) []string {
	if d == nil {
		return nil
	}

	switch d.(type) {
	case Definer, []Field, []FieldDefinition, Definitions, StaticDefinitions, []string:
		return fieldNames(d, exclude)
	}

	var (
		rTyp       = reflect.TypeOf(d)
		excludeMap = make(map[string]struct{})
	)
	if rTyp.Kind() == reflect.Ptr {
		rTyp = rTyp.Elem()
	}
	for _, name := range exclude {
		excludeMap[name] = struct{}{}
	}

	var (
		n     = rTyp.NumField()
		names = make([]string, 0, n)
	)
	for i := 0; i < n; i++ {
		var f = rTyp.Field(i)
		if _, ok := excludeMap[f.Name]; ok {
			continue
		}
		names = append(names, f.Name)
	}

	return names
}

const codeErrInitialised errors.GoCode = "notInitialised"

var errInitialised = errors.New(codeErrInitialised, "registry is not properly initialised yet")

// A shortcut to try and get a value without calling and having to set up the [Definitions].
//
// The modelInstance should be a pointer to your model.
//
// If the field exists, hasField will be true, but the field does not define a StructField
// suitable for retrieval without instantiating the [Definitions] first.
//
// If an empty string is provided as the fieldName, this function will use the primary
// field instead.
//
// The value will not be OK if:
// 1. The field is not present at all.
// 2. The structField was not provided.
// 3. Any path in the structfield index is nil (except for final.)
// 4. If the value implements the [Binder] interface.
func FastGet(modelInstance reflect.Value, fieldName string) (val reflect.Value, hasField bool, err error) {
	t := modelInstance.Type()
	modelInstance = modelInstance.Elem() // models should ALWAYS be pointers.

	if modelInstance.Kind() != reflect.Struct {
		panic("expected struct type for *modelMeta.FastGet")
	}

	m, ok := modelReg[t]
	if !ok {
		return val, true, errInitialised
	}

	if fieldName == "" {
		if m.definitions == nil {
			return val, true, errInitialised
		}

		fieldName = m.definitions.PrimaryField
	}

	if fieldName == "" {
		return val, true, errors.FieldNull.Wrap(
			"Fieldname not provided and could not be inferred",
		)
	}

	sf, ok := m.fieldsMap[fieldName]
	if !ok {
		return val, false, errors.FieldNotFound.Wrapf(
			"Field %q was not found in model %T", fieldName,
			modelInstance.Interface(),
		)
	}

	// fast path to know if the field is at least present in the model
	// fieldName will always be a valid mapkey if a field was defined.
	// if the field did not provide a structfield or it is nil, that is ok,
	// but at least now we know that the field exists without any extra work.
	if sf == nil {
		return val, true, errors.FieldNull.Wrapf(
			"Field %q in model %T does not allow for FastGet",
			fieldName, modelInstance.Interface(),
		)
	}

	// if type is binder type we cannot continue.
	// it requires too much setup, and that is out of scope
	// for this func.
	if sf.Type.Implements(_binder) {
		return val, true, errors.NotImplemented.Wrapf(
			"value %T for field %q in model %T implements Binder, cannot use FastGet",
			val.Interface(), fieldName, modelInstance.Interface(),
		)
	}

	val, err = modelInstance.FieldByIndexErr(sf.Index)
	return val, true, err
}

// SetPrimaryKey sets the primary key field of a Definer.
//
// If the primary key field is not found, this function will panic.
func SetPrimaryKey(ctx context.Context, d Definer, value interface{}) error {
	var f = Define(ctx, d).Primary()
	if f == nil {
		assert.Fail(
			"primary key not found in %T",
			d,
		)
	}

	return f.SetValue(value, false)
}

// PrimaryKey returns the primary key field of a Definer.
//
// If the primary key field is not found, this function will panic.
//
// This function can operate on the following types:
//
// 1. Definer (models)
// 2. Definitions
// 3. Fields
// 4. Any other value of reflect.Kind:
//   - reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64
//   - reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64
//   - reflect.String
//   - Byte slices will be converted
//
// If a value implements [driver.Valuer], it's Value method will be called.
//
// If a value implements [fmt.Stringer] and none of the previous kinds matched,
// we will call the [fmt.Stringer.String] method on said value.
func PrimaryKey(ctx context.Context, obj any) interface{} {
	var (
		f   Field
		val any
		rT  reflect.Type
		rV  reflect.Value
	)

typeSwitch:
	switch v := obj.(type) {
	case Definer:

		if len(modelReg) == 0 {
			val = Define(ctx, v).Primary().GetValue()
			break typeSwitch
		}

		pk, exists, err := FastGet(reflect.ValueOf(obj), "")
		if !exists {
			panic("primary field does not exist")
		}

		if errors.FieldNull.Is(err) || errInitialised.Is(err) {
			val = Define(ctx, v).Primary().GetValue()
			break typeSwitch
		}

		if err != nil {
			panic("primary field value is not valid or suitable for use with PrimaryKey: " + err.Error())
		}

		rT = pk.Type()
		rV = pk

	case Field:
		f = v
		val = f.GetValue()

	case Definitions:
		f = v.Primary()
		val = f.GetValue()

	default:
		val = v
	}

	if rT == nil {
		rT = reflect.TypeOf(val)
		rV = reflect.ValueOf(val)
	}

typeCheck:
	for rV.IsValid() {
		switch {
		case rT.Implements(_DRIVER_VALUE):
			v, err := rV.Interface().(driver.Valuer).Value()
			if err != nil {
				panic("primary field value is not valid or suitable for use with PrimaryKey: " + err.Error())
			}
			rV = reflect.ValueOf(v)
			rT = rV.Type()

		case rT.Kind() == reflect.Pointer || rT.Kind() == reflect.Interface:
			rT = rT.Elem()
			rV = rV.Elem()

		default:
			break typeCheck
		}
	}

	// return uint64 for both unsigned and signed integer types.
	switch rT.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return uint64(rV.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return rV.Uint()
	case reflect.String:
		return rV.String()
	}

	if rV.Kind() == reflect.Slice && rV.Type().Elem().Kind() == reflect.Uint8 {
		// Convert []byte to string for easier comparison and usage as a key
		rV = rV.Convert(reflect.TypeFor[string]())
		return rV.String()
	}

	switch v := val.(type) {
	case fmt.Stringer:
		return v.String()
	}

	return val
}

// SetMany sets multiple fields on a Definer.
//
// The values parameter is a map where the keys are the names of the fields to set.
//
// The values must be of the correct type for the fields.
func SetMany(ctx context.Context, d Definer, values map[string]interface{}) error {
	for name, value := range values {
		if err := assert.Err(set(Define(ctx, d), name, value, false)); err != nil {
			return err
		}
	}
	return nil
}

// Set sets the value of a field on a Definer.
//
// If the field is not found, the value is not of the correct type or another constraint is violated, this function will panic.
//
// If the field is marked as non editable, this function will panic.
func Set(ctx context.Context, d Definer, name string, value interface{}) error {
	return set(Define(ctx, d), name, value, false)
}

// ForceSet sets the value of a field on a Definer.
//
// If the field is not found, the value is not of the correct type or another constraint is violated, this function will panic.
//
// This function will allow setting the value of a field that is marked as not editable.
func ForceSet(ctx context.Context, d Definer, name string, value interface{}) error {
	return set(Define(ctx, d), name, value, true)
}

// Get retrieves the value of a field on a Definer.
//
// If the field is not found, this function will panic.
//
// Type assertions are used to ensure that the value is of the correct type,
// as well as providing less work for the caller.
func Get[T any](ctx context.Context, d any, name string) T {
	var defs Definitions

	switch d := d.(type) {
	case Definer:
		defs = Define(ctx, d)
	case Definitions:
		defs = d
	default:
		assert.Fail(
			"get (%T): expected Definer or Definitions, got %T",
			d, d,
		)
		return *(new(T))
	}

	var f, ok = defs.Field(name)
	if !ok {

		var method, ok = Method[T](d, name)
		if ok {
			return method
		}

		assert.Fail(
			"get (%T): no field named %q",
			d, name,
		)
	}

	var v = f.GetValue()
	switch t := v.(type) {
	case T:
		return t
	case *T:
		return *t
	case nil:
		return *(new(T))
		//	default:
		//		assert.Fail(
		//			"get (%T): field %q is not of type %T",
		//			d, name, v,
		//		)
	}

	var (
		n       T
		resultT = reflect.TypeOf(n)
		resultV = reflect.New(resultT)
		rVal    = reflect.ValueOf(v)
	)

	if rVal.Type().AssignableTo(resultT) {
		resultV.Elem().Set(rVal)
		return resultV.Elem().Interface().(T)
	}

	if rVal.Type().ConvertibleTo(resultT) {
		resultV.Elem().Set(rVal.Convert(resultT))
		return resultV.Elem().Interface().(T)
	}

	assert.Fail(
		"get (%T): field %q is not of type %T, got %T",
		d, name, resultT, rVal.Type(),
	)
	return *(new(T))
}

type Function interface{}

// Method retrieves a method from an object.
//
// The generic type parameter must be the type of the method.
func Method[T Function](obj interface{}, name string) (n T, ok bool) {
	var m, err = django_reflect.Method[T](obj, name)
	return m, err == nil
}

func set(d Definitions, name string, value interface{}, force bool) error {
	var f, ok = d.Field(name)
	if !ok {
		var fieldNames = fieldNames(d, nil)
		return assert.Fail(
			fmt.Sprintf("set (%T): no field named %q in %+v", d, name, fieldNames),
		)
	}

	return f.SetValue(value, force)
}
