package attrs

import (
	"fmt"
	"reflect"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/elliotchance/orderedmap/v2"
)

type ObjectDefinitions struct {
	Object       Definer
	PrimaryField string
	Table        string
	ObjectFields *orderedmap.OrderedMap[string, Field]
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
// - func() Field: a function that returns a field
// - func() (Field, error): a function that returns a field and an error
// - func() []Field: a function that returns a slice of fields
// - func() ([]Field, error): a function that returns a slice of fields and an error
// - func(d Definer) Field: a function that takes a Definer and returns a field
// - func(d Definer) (Field, error): a function that takes a Definer and returns a field and an error
// - func(d Definer) []Field: a function that takes a Definer and returns a slice of fields
// - func(d Definer) ([]Field, error): a function that takes a Definer and returns a slice of fields and an error
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

		// func() (field, ?error)
		case func() Field:
			fld = v()
		case func() (Field, error):
			fld, err = v()

		// func() ([]field, ?error)
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

// Define creates a new object definitions.
//
// This can then be returned by the FieldDefs method of a model
// to make it comply with the Definer interface.
//
// For information about the arguments, see the
// [UnpackFieldsFromArgs] function.
func Define[T1 Definer, T2 any](d T1, fieldDefinitions ...T2) *ObjectDefinitions {
	var fields, err = UnpackFieldsFromArgs(d, fieldDefinitions...)
	if err != nil {
		assert.Fail("define (%T): %v", d, err)
	}

	var defs = &ObjectDefinitions{
		Object: d,
	}

	var primaryField string
	var m = orderedmap.NewOrderedMap[string, Field]()
	for _, f := range fields {

		if f.IsPrimary() && primaryField == "" {
			primaryField = f.Name()
		}

		if binder, ok := f.(UnboundField); ok {
			var err error
			f, err = binder.BindField(d)
			if err != nil {
				assert.Fail("bind (%T): %v", d, err)
			}
		}

		f.BindToDefinitions(defs)

		m.Set(f.Name(), f)
	}

	defs.ObjectFields = m
	defs.PrimaryField = primaryField
	return defs
}

func (d *ObjectDefinitions) SignalChange(f Field, value interface{}) {
	if m, ok := d.Object.(CanSignalChanged); ok {
		m.SignalChange(f, value)
	}
}

func (d *ObjectDefinitions) TableName() string {
	if d.Table != "" {
		return d.Table
	}
	var rTyp = reflect.TypeOf(d.Object)
	if rTyp.Kind() == reflect.Ptr {
		rTyp = rTyp.Elem()
	}
	var tableName = toSnakeCase(rTyp.Name())
	return tableName
}

func (d *ObjectDefinitions) WithTableName(name string) *ObjectDefinitions {
	d.Table = name
	return d
}

func (d *ObjectDefinitions) Set(name string, value interface{}) error {
	return set(d, name, value, false)
}

func (d *ObjectDefinitions) ForceSet(name string, value interface{}) error {
	return set(d, name, value, true)
}

func (d *ObjectDefinitions) Get(name string) interface{} {
	var f, ok = d.ObjectFields.Get(name)
	if !ok {
		return assert.Fail(
			"get (%T): field %q not found in %T",
			d.Object, name, d.Object,
		)
	}
	return f.GetValue()
}

func (d *ObjectDefinitions) Field(name string) (f Field, ok bool) {
	f, ok = d.ObjectFields.Get(name)
	return
}

func (d *ObjectDefinitions) Fields() []Field {
	var m = make([]Field, d.ObjectFields.Len())
	var i = 0
	for head := d.ObjectFields.Front(); head != nil; head = head.Next() {
		m[i] = head.Value
		i++
	}
	return m
}

func (d *ObjectDefinitions) Primary() Field {
	if d.PrimaryField == "" {
		return nil
	}
	f, _ := d.ObjectFields.Get(d.PrimaryField)
	return f
}

func (d *ObjectDefinitions) Instance() Definer {
	if d.Object == nil {
		var rTyp = reflect.TypeOf(d)
		if rTyp == nil || rTyp.Kind() == reflect.Invalid {
			return nil
		}
		return NewObject[Definer](d.Object)
	}
	return d.Object
}

func (d *ObjectDefinitions) Len() int {
	return d.ObjectFields.Len()
}
