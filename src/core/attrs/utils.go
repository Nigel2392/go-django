package attrs

import (
	"fmt"
	"iter"
	"reflect"

	"github.com/Nigel2392/go-django/src/utils/mixins"
)

// DefinerList converts a slice of []T where the underlying type is of type Definer to []Definer.
func DefinerList[T Definer](list []T) []Definer {
	var n = len(list)
	if n == 0 {
		return nil
	}
	var l = make([]Definer, n)
	for i, v := range list {
		l[i] = v
	}
	return l
}

// UnpackFieldsFromArgs unpacks the fields from the given arguments.
//
// The fields are passed as variadic arguments, and can be of many types:
//
// - Field: a field (or any type that implements the Field interface)
// - []Field: a slice of fields
// - UnboundFieldConstruuctor: a constructor for a field that needs to be bound
// - []UnboundFieldConstructor: a slice of unbound field constructors
// - UnboundField: an unbound field that needs to be bound
// - []UnboundField: a slice of unbound fields that need to be bound
// - func() []any: a function of which the result will be recursively unpacked
// - func() Field: a function that returns a field
// - func() (Field, error): a function that returns a field and an error
// - func() []Field: a function that returns a slice of fields
// - func() ([]Field, error): a function that returns a slice of fields and an error
// - func(d Definer) []any: a function that takes a Definer and returns a slice of any to be recursively unpacked
// - func(d Definer) Field: a function that takes a Definer and returns a field
// - func(d Definer) (Field, error): a function that takes a Definer and returns a field and an error
// - func(d Definer) []Field: a function that takes a Definer and returns a slice of fields
// - func(d Definer) ([]Field, error): a function that takes a Definer and returns a slice of fields and an error
// - func(d T1) []any: a function that takes a Definer of type T1 and returns a slice of any to be recursively unpacked
// - func(d T1) Field: a function that takes a Definer of type T1 and returns a field
// - func(d T1) (Field, error): a function that takes a Definer of type T1 and returns a field and an error
// - func(d T1) []Field: a function that takes a Definer of type T1 and returns a slice of fields
// - func(d T1) ([]Field, error): a function that takes a Definer of type T1 and returns a slice of fields and an error
// - string: a field name, which will be converted to a Field with no configuration
func UnpackFieldsFromArgs[T1 Definer, T2 any](definer T1, args ...T2) ([]Field, error) {
	var fields = make([]Field, 0, len(args))
	for _, f := range args {
		var (
			val  any = f
			flds []Field
			fld  Field
			err  error
		)
		switch v := val.(type) {
		case Field:
			fld = v
		case []Field:
			flds = v

		case UnboundFieldConstructor:
			fld, err = v.BindField(definer)
		case []UnboundFieldConstructor:
			flds = make([]Field, len(v))
			for i, u := range v {
				flds[i], err = u.BindField(definer)
				if err != nil {
					return nil, fmt.Errorf(
						"fieldsFromArgs (%T): %v",
						definer, err,
					)
				}
			}

		case []UnboundField:
			flds = make([]Field, len(v))
			for i, u := range v {
				flds[i] = u
			}

		case []any:
			var unpacked, err = UnpackFieldsFromArgs(definer, v...)
			if err != nil {
				return nil, fmt.Errorf(
					"fieldsFromArgs (%T): %v",
					definer, err,
				)
			}
			flds = unpacked

		// func() (field, ?error)
		case func() Field:
			fld = v()
		case func() (Field, error):
			fld, err = v()

		// func() ([]field, ?error)
		case func() []any:
			var unpacked, err = UnpackFieldsFromArgs(definer, v()...)
			if err != nil {
				return nil, fmt.Errorf(
					"fieldsFromArgs (%T): %v",
					definer, err,
				)
			}
			flds = unpacked
		case func() []Field:
			flds = v()
		case func() ([]Field, error):
			flds, err = v()

		// func(t1) (field, ?error)
		case func(d T1) Field:
			fld = v(definer)
		case func(d T1) (Field, error):
			fld, err = v(definer)

		// func(t1) ([]field, ?error)
		case func(d T1) []any:
			var unpacked, err = UnpackFieldsFromArgs(definer, v(definer)...)
			if err != nil {
				return nil, fmt.Errorf(
					"fieldsFromArgs (%T): %v",
					definer, err,
				)
			}
			flds = unpacked
		case func(d T1) []Field:
			flds = v(definer)
		case func(d T1) ([]Field, error):
			flds, err = v(definer)

		// func(d Definer) (field, ?error)
		case func(d Definer) Field:
			fld = v(definer)
		case func(d Definer) (Field, error):
			fld, err = v(definer)

		// func(d Definer) ([]field, ?error)
		case func(d Definer) []any:
			var unpacked, err = UnpackFieldsFromArgs(definer, v(definer)...)
			if err != nil {
				return nil, fmt.Errorf(
					"fieldsFromArgs (%T): %v",
					definer, err,
				)
			}
			flds = unpacked
		case func(d Definer) []Field:
			flds = v(definer)
		case func(d Definer) ([]Field, error):
			flds, err = v(definer)

		case string:
			fld = NewField(definer, v, nil)

		default:
			return nil, fmt.Errorf(
				"fieldsFromArgs (%T): unsupported field type %T",
				definer, f,
			)
		}

		if err != nil {
			return nil, fmt.Errorf(
				"fieldsFromArgs (%T): %v",
				definer, err,
			)
		}
		if fld != nil {
			fields = append(fields, fld)
		}
		if len(flds) > 0 {
			fields = append(fields, flds...)
		}

	}
	return fields, nil
}

// UnpackFieldsFromArgsIter unpacks the fields from the given arguments.
//
// It returns an iterator that yields fields and errors.
//
// The fields are passed as variadic arguments, and can be of many types:
//
// - Field: a field (or any type that implements the Field interface)
// - []Field: a slice of fields
// - UnboundFieldConstruuctor: a constructor for a field that needs to be bound
// - []UnboundFieldConstructor: a slice of unbound field constructors
// - UnboundField: an unbound field that needs to be bound
// - []UnboundField: a slice of unbound fields that need to be bound
// - iter.Seq2[Field, error]: iterators to possibly increase performance
// - func() iter.Seq2[Field, error]: iterators to possibly increase performance
// - func(Definer) iter.Seq2[Field, error]: iterators to possibly increase performance
// - func() []any: a function of which the result will be recursively unpacked
// - func() Field: a function that returns a field
// - func() (Field, error): a function that returns a field and an error
// - func() []Field: a function that returns a slice of fields
// - func() ([]Field, error): a function that returns a slice of fields and an error
// - func(d Definer) []any: a function that takes a Definer and returns a slice of any to be recursively unpacked
// - func(d Definer) Field: a function that takes a Definer and returns a field
// - func(d Definer) (Field, error): a function that takes a Definer and returns a field and an error
// - func(d Definer) []Field: a function that takes a Definer and returns a slice of fields
// - func(d Definer) ([]Field, error): a function that takes a Definer and returns a slice of fields and an error
// - func(d T1) []any: a function that takes a Definer of type T1 and returns a slice of any to be recursively unpacked
// - func(d T1) Field: a function that takes a Definer of type T1 and returns a field
// - func(d T1) (Field, error): a function that takes a Definer of type T1 and returns a field and an error
// - func(d T1) []Field: a function that takes a Definer of type T1 and returns a slice of fields
// - func(d T1) ([]Field, error): a function that takes a Definer of type T1 and returns a slice of fields and an error
// - string: a field name, which will be converted to a Field with no configuration
func UnpackFieldsFromArgsIter[T1 Definer, T2 any](definer T1, args ...T2) iter.Seq2[Field, error] {
	return func(yield func(Field, error) bool) {
		unpackFieldsFromArgsIter(yield, definer, args)
	}
}

func yieldIter[T any](yield func(T, error) bool, iterator iter.Seq2[T, error]) bool {
	for v, err := range iterator {
		if err != nil {
			if !yield(v, err) {
				return false
			}
		}
		if !yield(v, nil) {
			return false
		}
	}
	return true
}

func yieldMultiple[T any](yield func(T, error) bool, err error, items []T) bool {
	if err != nil {
		if !yield(*new(T), err) {
			return false
		}
		return true
	}
	for _, item := range items {
		if !yield(item, nil) {
			return false
		}
	}
	return true
}

func unpackFieldsFromArgsIter[T1 Definer, T2 any](yield func(Field, error) bool, definer T1, args []T2) bool {
	for _, f := range args {
		switch v := any(f).(type) {
		case Field:
			if !yield(v, nil) {
				return false
			}
		case []Field:
			if !yieldMultiple(yield, nil, v) {
				return false
			}

		case UnboundFieldConstructor:
			var fld, err = v.BindField(definer)
			if err != nil {
				yield(nil, fmt.Errorf(
					"fieldsFromArgs (%T): %v",
					definer, err,
				))
				return false
			}
			if !yield(fld, nil) {
				return false
			}
		case []UnboundFieldConstructor:
			for _, u := range v {
				var fld, err = u.BindField(definer)
				if err != nil {
					yield(nil, fmt.Errorf(
						"fieldsFromArgs (%T): %v",
						definer, err,
					))
					return false
				}

				if !yield(fld, nil) {
					return false
				}
			}

		case []UnboundField:
			for _, u := range v {
				var fld, err = u.BindField(definer)
				if err != nil {
					yield(nil, fmt.Errorf(
						"fieldsFromArgs (%T): %v",
						definer, err,
					))
					return false
				}

				if !yield(fld, nil) {
					return false
				}
			}

		case []any:
			if !unpackFieldsFromArgsIter(yield, definer, v) {
				return false
			}

		case iter.Seq2[Field, error]:
			if !yieldIter(yield, v) {
				return false
			}

		case func() iter.Seq2[Field, error]:
			if !yieldIter(yield, v()) {
				return false
			}

		case func(Definer) iter.Seq2[Field, error]:
			if !yieldIter(yield, v(definer)) {
				return false
			}

		// func() (field, ?error)
		case func() Field:
			if !yield(v(), nil) {
				return false
			}
		case func() (Field, error):
			var fld, err = v()
			if !yield(fld, err) {
				return false
			}

		// func() ([]field, ?error)
		case func() []any:
			if !unpackFieldsFromArgsIter(yield, definer, v()) {
				return false
			}

		// func(t1) (field, ?error)
		case func(d T1) Field:
			if !yield(v(definer), nil) {
				return false
			}

		case func(d T1) (Field, error):
			var fld, err = v(definer)
			if err != nil {
				yield(nil, fmt.Errorf(
					"fieldsFromArgs (%T): %v",
					definer, err,
				))
				return false
			}

			if !yield(fld, nil) {
				return false
			}

		// func(t1) ([]field, ?error)
		case func(d T1) []any:
			if !unpackFieldsFromArgsIter(yield, definer, v(definer)) {
				return false
			}

		case func(d T1) []Field:
			if !yieldMultiple(yield, nil, v(definer)) {
				return false
			}

		case func(d T1) ([]Field, error):
			var flds, err = v(definer)
			if !yieldMultiple(yield, err, flds) {
				return false
			}

		// func(d Definer) (field, ?error)
		case func(d Definer) Field:
			if !yield(v(definer), nil) {
				return false
			}
		case func(d Definer) (Field, error):
			var fld, err = v(definer)
			if err != nil {
				yield(nil, fmt.Errorf(
					"fieldsFromArgs (%T): %v",
					definer, err,
				))
				return false
			}
			if !yield(fld, nil) {
				return false
			}

		// func(d Definer) ([]field, ?error)
		case func(d Definer) []any:
			if !unpackFieldsFromArgsIter(yield, definer, v(definer)) {
				return false
			}

		case func(d Definer) []Field:
			if !yieldMultiple(yield, nil, v(definer)) {
				return false
			}

		case func(d Definer) ([]Field, error):
			var flds, err = v(definer)
			if err != nil {
				yield(nil, fmt.Errorf(
					"fieldsFromArgs (%T): %v",
					definer, err,
				))
				return false
			}
			if !yieldMultiple(yield, err, flds) {
				return false
			}

		case string:
			if !yield(NewField(definer, v, nil), nil) {
				return false
			}

		default:
			yield(nil, fmt.Errorf(
				"fieldsFromArgs (%T): unsupported field type %T",
				definer, f,
			))
			return false
		}
	}
	return true
}

func structFieldsMixinFunc[T any](fn func(obj T, depth int, field reflect.StructField, value reflect.Value) (reflect.Value, bool)) func(obj T, depth int) iter.Seq[T] {
	var _T = reflect.TypeOf((*T)(nil)).Elem()
	return func(obj T, depth int) iter.Seq[T] {
		var rVal = reflect.ValueOf(obj)
		var rTyp = rVal.Type()
		if rTyp.Kind() == reflect.Ptr {
			rTyp = rTyp.Elem()
			rVal = rVal.Elem()
		}
		if rTyp.Kind() != reflect.Struct {
			return mixins.NillSeq
		}
		return func(yield func(T) bool) {
			for i := 0; i < rTyp.NumField(); i++ {
				var (
					fieldT = rTyp.Field(i)
					fieldV = rVal.Field(i)
				)

				if fieldV.Kind() == reflect.Ptr && fieldV.IsNil() {
					continue
				}

				var typ = fieldT.Type
				switch {
				case typ == _T:
				case _T.Kind() == reflect.Interface && typ.Implements(_T):
				case _T.Kind() == reflect.Interface && fieldV.CanAddr() && fieldV.Addr().Type().Implements(_T):
					fieldV = fieldV.Addr()
				default:
					continue
				}

				fieldV, ok := fn(obj, depth, fieldT, fieldV)
				if !ok {
					continue
				}

				if mv, ok := fieldV.Interface().(T); ok {
					if !yield(mv) {
						return
					}
				}
			}
		}
	}
}

type mixinDefiner interface {
	IsModelMixin()
}

var _modelMixinT = reflect.TypeOf((*mixinDefiner)(nil)).Elem()

func structFieldsMixinFuncCheck[T any](obj T, depth int, field reflect.StructField, value reflect.Value) (reflect.Value, bool) {
	if !field.Anonymous {
		return value, false
	}

	if !field.IsExported() {
		return value, false
	}

	if field.Type.Implements(_modelMixinT) {
		return value, true
	}

	if field.Type.Kind() != reflect.Ptr && !(value.CanAddr() || value.Addr().Type().Implements(_modelMixinT)) {
		return value, false
	}

	return value.Addr(), true
}

func ModelMixins(obj Definer, topdown bool) iter.Seq2[any, int] {
	var iter = mixins.MixinsFunc[any](obj, topdown, structFieldsMixinFunc(structFieldsMixinFuncCheck[any]))
	return func(yield func(any, int) bool) {
		for m, depth := range iter {
			if depth == 0 {
				continue
			}

			if !yield(m, depth) {
				return
			}
		}
	}
}
