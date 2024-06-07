package attrs

import "github.com/elliotchance/orderedmap/v2"

type ObjectDefinitions struct {
	Object       Definer
	PrimaryField string
	ObjectFields *orderedmap.OrderedMap[string, Field]
}

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

func (d *ObjectDefinitions) Primary() Field {
	if d.PrimaryField == "" {
		return nil
	}
	f, _ := d.ObjectFields.Get(d.PrimaryField)
	return f
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
