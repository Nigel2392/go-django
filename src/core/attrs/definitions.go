package attrs

import (
	"iter"
	"reflect"

	"github.com/Nigel2392/go-django/src/core/assert"
)

type ObjectDefinitions struct {
	objType      reflect.Type
	Object       Definer
	PrimaryField string
	Table        string
	ObjectFields *FieldsMap
}

type FieldsMap struct {
	ref   *ObjectDefinitions
	keys  []string
	items map[string]Field
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

	if _, ok := d.items[name]; ok {
		replaced = true
	} else {
		d.keys = append(d.keys, name)
	}

	d.items[name] = f
	return
}

func (d *FieldsMap) Get(name string) (f Field, ok bool) {
	f, ok = d.items[name]
	return f, ok
}

func (d *FieldsMap) Len() int {
	return len(d.items)
}

func (d *FieldsMap) Iter() iter.Seq2[string, Field] {
	return func(yield func(string, Field) bool) {
		for _, name := range d.keys {
			if !yield(name, d.items[name]) {
				return
			}
		}
	}
}

// Define creates a new object definitions.
//
// This can then be returned by the FieldDefs method of a model
// to make it comply with the Definer interface.
//
// For information about the arguments, see the
// [UnpackFieldsFromArgs] function.
func Define[T1 Definer, T2 any](d T1, fieldDefinitions ...T2) *ObjectDefinitions {

	var rt = reflect.TypeOf(d).Elem()
	var numField = rt.NumField()
	var defs = &ObjectDefinitions{
		objType: rt,
		Object:  d,
	}
	defs.ObjectFields = &FieldsMap{
		ref:   defs,
		items: make(map[string]Field, numField),
		keys:  make([]string, 0, numField),
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

	//for mixin, depth := range ModelMixins(d, true) {
	//	// Skip the first level, this is the model itself.
	//	if depth == 0 {
	//		continue
	//	}
	//
	//	binder, ok := mixin.(Embedded)
	//	if ok {
	//		if err := binder.BindToEmbedder(d); err != nil {
	//			assert.Fail("bind embedded (%T): %v", d, err)
	//		}
	//	}
	//
	//	if unpacker, ok := mixin.(FieldUnpackerMixin); ok {
	//		if err := unpacker.ObjectFields(d, defs.ObjectFields); err != nil {
	//			assert.Fail("unpack fields (%T): %v", d, err)
	//		}
	//	}
	//}

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
	return ColumnName(d.objType.Name())
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
	var m = make([]Field, len(d.ObjectFields.keys))
	for i, name := range d.ObjectFields.keys {
		m[i] = d.ObjectFields.items[name]
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
		if d.objType == nil || d.objType.Kind() == reflect.Invalid {
			return nil
		}
		return NewObject[Definer](d.Object)
	}
	return d.Object
}

func (d *ObjectDefinitions) Len() int {
	return d.ObjectFields.Len()
}
