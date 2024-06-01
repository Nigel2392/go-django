package attrs

import (
	"reflect"

	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/tags"
)

const ATTR_TAG_NAME = "attrs"

type FieldConfig struct {
	Null     bool
	Blank    bool
	ReadOnly bool
	Label    string
	HelpText string
}

func autoDefinitionStructTag(t reflect.StructField) FieldConfig {
	var (
		tag  = t.Tag.Get(ATTR_TAG_NAME)
		data = FieldConfig{}
	)

	var tagMap = tags.ParseTags(tag)
	for k, v := range tagMap {
		switch k {
		case "null":
			data.Null = true
		case "blank":
			data.Blank = true
		case "readonly":
			data.ReadOnly = true
		case "label":
			data.Label = v[0]
		case "helpText":
			data.HelpText = v[0]
		}
	}

	return data
}

func AutoDefinitions[T Definer](instance T, include ...string) Definitions {
	var m = make([]Field, 0)

	var (
		instance_t_ptr = reflect.TypeOf(instance)
		instance_v_ptr = reflect.ValueOf(instance)
	)

	assert.Equal(
		instance_t_ptr.Kind(), reflect.Ptr,
		"instance %T must be a pointer", instance,
	)

	var (
		instance_t = instance_t_ptr.Elem()
		instance_v = instance_v_ptr.Elem()
	)

	if len(include) == 0 {
		for i := 0; i < instance_t.NumField(); i++ {
			var (
				field_t = instance_t.Field(i)
				field_v = instance_v.Field(i)
				attrs   = autoDefinitionStructTag(field_t)
			)

			var skip = (field_t.Anonymous ||
				field_t.PkgPath != "" ||
				field_t.Tag.Get(ATTR_TAG_NAME) == "-")

			if skip {
				continue
			}

			m = append(m, &FieldDef{
				attrDef:        attrs,
				instance_t_ptr: instance_t_ptr,
				instance_v_ptr: instance_v_ptr,
				instance_t:     instance_t,
				instance_v:     instance_v,
				field_t:        field_t,
				field_v:        field_v,
			})
		}
	} else {
		for _, name := range include {
			var field_t, ok = instance_t.FieldByName(name)

			assert.True(ok, "field %q not found in %T", name, instance)

			var (
				attrs   = autoDefinitionStructTag(field_t)
				field_v = instance_v.FieldByIndex(field_t.Index)
			)

			var skip = (field_t.Anonymous ||
				field_t.PkgPath != "" ||
				field_t.Tag.Get(ATTR_TAG_NAME) == "-")

			if skip {
				continue
			}

			m = append(m, &FieldDef{
				attrDef:        attrs,
				instance_t_ptr: instance_t_ptr,
				instance_v_ptr: instance_v_ptr,
				instance_t:     instance_t,
				instance_v:     instance_v,
				field_t:        field_t,
				field_v:        field_v,
			})
		}
	}

	return Define(instance, m...)
}
