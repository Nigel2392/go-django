package attrs

type ObjectDefinitions struct {
	Object       Definer
	ObjectFields map[string]Field
}

func Define(d Definer, fieldDefinitions map[string]Field) *ObjectDefinitions {
	return &ObjectDefinitions{
		Object:       d,
		ObjectFields: fieldDefinitions,
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
	f, ok = d.ObjectFields[name]
	return
}
