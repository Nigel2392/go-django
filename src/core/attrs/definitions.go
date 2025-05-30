package attrs

import (
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

// Define creates a new object definitions.
//
// This can then be returned by the FieldDefs method of a model
// to make it comply with the Definer interface.
func Define(d Definer, fieldDefinitions ...Field) *ObjectDefinitions {
	var primaryField string

	var m = orderedmap.NewOrderedMap[string, Field]()
	for _, f := range fieldDefinitions {

		if f.IsPrimary() && primaryField == "" {
			primaryField = f.Name()
		}

		m.Set(f.Name(), f)
	}

	return &ObjectDefinitions{
		Object:       d,
		ObjectFields: m,
		PrimaryField: primaryField,
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
		var objectT = reflect.TypeOf(d.Object)
		if objectT.Kind() == reflect.Ptr {
			objectT = objectT.Elem()
		}
		return reflect.New(objectT).Interface().(Definer)
	}
	return d.Object
}

func (d *ObjectDefinitions) Len() int {
	return d.ObjectFields.Len()
}
