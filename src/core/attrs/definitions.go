package attrs

import (
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
	return d.Table
}

func (d *ObjectDefinitions) WithTableName(name string) *ObjectDefinitions {
	d.Table = name
	return d
}

func (d *ObjectDefinitions) Set(name string, value interface{}) error {
	return set(d.Object, name, value, false)
}

func (d *ObjectDefinitions) ForceSet(name string, value interface{}) error {
	return set(d.Object, name, value, true)
}

func (d *ObjectDefinitions) Get(name string) interface{} {
	return Get[interface{}](d.Object, name)
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
	return d.Object
}

func (d *ObjectDefinitions) Len() int {
	return d.ObjectFields.Len()
}
