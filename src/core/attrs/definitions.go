package attrs

import (
	"reflect"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/utils/mixins"
	"github.com/elliotchance/orderedmap/v2"
)

type ObjectDefinitions struct {
	Object       Definer
	PrimaryField string
	Table        string
	ObjectFields *FieldsMap
}

type FieldsMap struct {
	ref *ObjectDefinitions
	*orderedmap.OrderedMap[string, Field]
}

func (d *FieldsMap) Set(name string, f Field) (replaced bool) {
	if binder, ok := f.(UnboundField); ok {
		var err error
		f, err = binder.BindField(d.ref.Object)
		if err != nil {
			assert.Fail("bind (%T): %v", d, err)
		}
	}

	f.BindToDefinitions(d.ref)

	return d.OrderedMap.Set(name, f)
}

// Define creates a new object definitions.
//
// This can then be returned by the FieldDefs method of a model
// to make it comply with the Definer interface.
//
// For information about the arguments, see the
// [UnpackFieldsFromArgs] function.
func Define[T1 Definer, T2 any](d T1, fieldDefinitions ...T2) *ObjectDefinitions {

	var defs = &ObjectDefinitions{
		Object: d,
	}
	defs.ObjectFields = &FieldsMap{
		ref:        defs,
		OrderedMap: orderedmap.NewOrderedMap[string, Field](),
	}

	var fieldsIter = UnpackFieldsFromArgsIter(d, fieldDefinitions...)
	for f, err := range fieldsIter {
		if err != nil {
			assert.Fail("define (%T): %v", d, err)
			continue
		}

		if f.IsPrimary() && defs.PrimaryField == "" {
			defs.PrimaryField = f.Name()
		}

		defs.ObjectFields.Set(f.Name(), f)
	}

	for mixin, depth := range mixins.Mixins[any](d, true) {
		// Skip the first level, this is the model itself.
		if depth == 0 {
			continue
		}

		binder, ok := mixin.(Embedded)
		if ok {
			if err := binder.BindToEmbedder(d); err != nil {
				assert.Fail("bind embedded (%T): %v", d, err)
			}
		}

		if unpacker, ok := mixin.(FieldUnpackerMixin); ok {
			if err := unpacker.ObjectFields(d, defs.ObjectFields); err != nil {
				assert.Fail("unpack fields (%T): %v", d, err)
			}
		}
	}

	return defs
}

func (d *ObjectDefinitions) SignalChange(f Field, value interface{}) {
	if m, ok := d.Object.(CanSignalChanged); ok {
		m.SignalChange(f, value)
	}
}

func (d *ObjectDefinitions) SignalReset(f Field) {
	if m, ok := d.Object.(CanSignalChanged); ok {
		m.SignalReset(f)
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
	var tableName = ColumnName(rTyp.Name())
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
