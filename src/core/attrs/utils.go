package attrs

import (
	"fmt"
	"iter"
	"net/mail"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/Nigel2392/go-django/src/core/contenttypes"
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

// ToString converts a value to a string.
//
// This should be the human-readable representation of the value.
//
// If the value is a struct with a content type, it will use the content type's InstanceLabel method to convert it to a string.
//
// time.Time, mail.Address, and error types are handled specially.
//
// If the value is a slice or array, it will convert each element to a string and join them with ", ".
//
// If all else fails, it will use fmt.Sprintf to convert the value to a string.
func ToString(v any) string {
	if v == nil {
		return ""
	}

	if s, ok := v.(string); ok {
		return s
	}

	var r = reflect.ValueOf(v)
	if r.Kind() == reflect.Ptr {
		r = r.Elem()
	}

	if r.Kind() == reflect.Struct {
		var cType = contenttypes.DefinitionForObject(
			v,
		)
		if cType != nil {
			return cType.InstanceLabel(v)
		}
	}

	return toString(r, v)
}

func toString(r reflect.Value, v any) string {
	switch v := v.(type) {
	case *mail.Address:
		return v.Address
	case time.Time:
		return v.Format(time.RFC3339)
	case fmt.Stringer:
		return v.String()
	case error:
		return v.Error()
	}

	if r.Kind() == reflect.Ptr {
		r = r.Elem()
	}

	switch r.Kind() {
	case reflect.String:
		return r.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(r.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(r.Uint(), 10)
	case reflect.Float32:
		return strconv.FormatFloat(r.Float(), 'f', -1, 32)
	case reflect.Float64:
		return strconv.FormatFloat(r.Float(), 'f', -1, 64)
	case reflect.Bool:
		return strconv.FormatBool(r.Bool())
	case reflect.Slice, reflect.Array:
		var b = make([]string, r.Len())
		for i := 0; i < r.Len(); i++ {
			b[i] = ToString(r.Index(i).Interface())
		}
		return strings.Join(b, ", ")
	case reflect.Struct:
		var cType = contenttypes.DefinitionForObject(
			v,
		)
		if cType != nil {
			return cType.InstanceLabel(v)
		}
	}

	return fmt.Sprintf("%v", v)
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
		var yieldMultiple = func(err error, fld []Field) bool {
			if err != nil {
				yield(nil, fmt.Errorf(
					"fieldsFromArgs (%T): %v",
					definer, err,
				))
				return false
			}

			for _, f := range fld {
				if !yield(f, nil) {
					return false
				}
			}
			return true
		}

		var yieldIter = func(iterator iter.Seq2[Field, error]) bool {
			for f, err := range iterator {
				if err != nil {
					yield(nil, fmt.Errorf(
						"fieldsFromArgs (%T): %v",
						definer, err,
					))
					return false
				}
				if !yield(f, nil) {
					return false
				}
			}
			return true
		}

		for _, f := range args {
			switch v := any(f).(type) {
			case Field:
				if !yield(v, nil) {
					return
				}
			case []Field:
				if !yieldMultiple(nil, v) {
					return
				}

			case UnboundFieldConstructor:
				var fld, err = v.BindField(definer)
				if err != nil {
					yield(nil, fmt.Errorf(
						"fieldsFromArgs (%T): %v",
						definer, err,
					))
					return
				}
				if !yield(fld, nil) {
					return
				}
			case []UnboundFieldConstructor:
				for _, u := range v {
					var fld, err = u.BindField(definer)
					if err != nil {
						yield(nil, fmt.Errorf(
							"fieldsFromArgs (%T): %v",
							definer, err,
						))
						return
					}

					if !yield(fld, nil) {
						return
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
						return
					}

					if !yield(fld, nil) {
						return
					}
				}

			case []any:
				var iterator = UnpackFieldsFromArgsIter(definer, v...)
				if !yieldIter(iterator) {
					return
				}

			case iter.Seq2[Field, error]:
				if !yieldIter(v) {
					return
				}

			case func() iter.Seq2[Field, error]:
				var iterator = v()
				if !yieldIter(iterator) {
					return
				}

			case func(Definer) iter.Seq2[Field, error]:
				var iterator = v(definer)
				if !yieldIter(iterator) {
					return
				}

			// func() (field, ?error)
			case func() Field:
				if !yield(v(), nil) {
					return
				}
			case func() (Field, error):
				var fld, err = v()
				if !yield(fld, err) {
					return
				}

			// func() ([]field, ?error)
			case func() []any:
				var iterator = UnpackFieldsFromArgsIter(definer, v()...)
				if !yieldIter(iterator) {
					return
				}

			// func(t1) (field, ?error)
			case func(d T1) Field:
				if !yield(v(definer), nil) {
					return
				}

			case func(d T1) (Field, error):
				var fld, err = v(definer)
				if err != nil {
					yield(nil, fmt.Errorf(
						"fieldsFromArgs (%T): %v",
						definer, err,
					))
					return
				}

				if !yield(fld, nil) {
					return
				}

			// func(t1) ([]field, ?error)
			case func(d T1) []any:
				var iterator = UnpackFieldsFromArgsIter(definer, v(definer)...)
				if !yieldIter(iterator) {
					return
				}

			case func(d T1) []Field:
				if !yieldMultiple(nil, v(definer)) {
					return
				}

			case func(d T1) ([]Field, error):
				var flds, err = v(definer)
				if !yieldMultiple(err, flds) {
					return
				}

			// func(d Definer) (field, ?error)
			case func(d Definer) Field:
				if !yield(v(definer), nil) {
					return
				}
			case func(d Definer) (Field, error):
				var fld, err = v(definer)
				if err != nil {
					yield(nil, fmt.Errorf(
						"fieldsFromArgs (%T): %v",
						definer, err,
					))
					return
				}
				if !yield(fld, nil) {
					return
				}

			// func(d Definer) ([]field, ?error)
			case func(d Definer) []any:
				var iterator = UnpackFieldsFromArgsIter(definer, v(definer)...)
				if !yieldIter(iterator) {
					return
				}

			case func(d Definer) []Field:
				if !yieldMultiple(nil, v(definer)) {
					return
				}

			case func(d Definer) ([]Field, error):
				var flds, err = v(definer)
				if err != nil {
					yield(nil, fmt.Errorf(
						"fieldsFromArgs (%T): %v",
						definer, err,
					))
					return
				}
				if !yieldMultiple(err, flds) {
					return
				}

			case string:
				if !yield(NewField(definer, v, nil), nil) {
					return
				}

			default:
				yield(nil, fmt.Errorf(
					"fieldsFromArgs (%T): unsupported field type %T",
					definer, f,
				))
				return
			}
		}
	}
}
