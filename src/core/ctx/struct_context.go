package ctx

import (
	"reflect"

	"github.com/Nigel2392/go-django/src/core/assert"
)

type StructContext struct {
	obj           interface{}
	data          map[string]any
	fields        map[string]reflect.Value
	DeniesContext bool
}

func NewStructContext(obj interface{}) Context {
	var t = reflect.TypeOf(obj)
	var v = reflect.ValueOf(obj)

	assert.Equal(
		t.Kind(), reflect.Ptr,
		"obj must be a pointer",
	)

	t = t.Elem()
	v = v.Elem()

	var fields = make(map[string]reflect.Value)
	for i := 0; i < t.NumField(); i++ {
		var field = t.Field(i)
		var name = field.Name
		fields[name] = v.Field(i)
	}

	return &StructContext{
		obj:    obj,
		data:   make(map[string]any),
		fields: fields,
	}
}

func (c *StructContext) Object() interface{} {
	return c.obj
}

func (c *StructContext) Set(key string, value any) {
	if v, ok := value.(Editor); ok {
		v.EditContext(key, c)
		return
	}

	if !c.DeniesContext {
		if context, ok := c.obj.(Context); ok {
			context.Set(key, value)
			return
		}
	}

	if field, ok := c.fields[key]; ok {
		var v = reflect.ValueOf(value)
		if v.Type() != field.Type() && v.Type().ConvertibleTo(field.Type()) {
			v = v.Convert(field.Type())
		}

		assert.Equal(v.Type(), field.Type(), "value type must be the same as field type")

		field.Set(v)
		return
	}

	c.data[key] = value
}

func (c *StructContext) Get(key string) any {
	if !c.DeniesContext && c.obj != nil {
		if context, ok := c.obj.(Context); ok {
			return context.Get(key)
		}
	}

	if field, ok := c.fields[key]; ok {
		return field.Interface()
	}

	return c.data[key]
}
