package attrs

import (
	"reflect"
	"strings"

	"github.com/Nigel2392/django/core/assert"
)

const ATTR_TAG_NAME = "attrs"

func autoDefinitionStructTag(t reflect.StructField) (null, blank, editable bool) {
	var (
		tag   = t.Tag.Get(ATTR_TAG_NAME)
		split = strings.Split(tag, "|")
	)
	null, blank, editable = false, false, true

	for _, s := range split {
		switch strings.TrimSpace(s) {
		case "null":
			null = true
		case "blank":
			blank = true
		case "readonly":
			editable = false
		}
	}
	return null, blank, editable
}

func AutoDefinitions[T Definer](instance T, include ...string) Definitions {
	var m = make(map[string]Field)

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
				field_t               = instance_t.Field(i)
				field_v               = instance_v.Field(i)
				name                  = field_t.Name
				null, blank, editable = autoDefinitionStructTag(field_t)
			)

			var skip = (field_t.Anonymous ||
				field_t.PkgPath != "" ||
				field_t.Tag.Get(ATTR_TAG_NAME) == "-")

			if skip {
				continue
			}

			m[name] = &FieldDef{
				Null:           null,
				Blank:          blank,
				Editable:       editable,
				instance_t_ptr: instance_t_ptr,
				instance_v_ptr: instance_v_ptr,
				instance_t:     instance_t,
				instance_v:     instance_v,
				field_t:        field_t,
				field_v:        field_v,
			}
		}
	} else {
		for _, name := range include {
			var field_t, ok = instance_t.FieldByName(name)

			assert.True(ok, "field %q not found in %T", name, instance)

			var (
				null, blank, editable = autoDefinitionStructTag(field_t)
				field_v               = instance_v.FieldByIndex(field_t.Index)
			)

			var skip = (field_t.Anonymous ||
				field_t.PkgPath != "" ||
				field_t.Tag.Get(ATTR_TAG_NAME) == "-")

			if skip {
				continue
			}

			m[name] = &FieldDef{
				Null:           null,
				Blank:          blank,
				Editable:       editable,
				instance_t_ptr: instance_t_ptr,
				instance_v_ptr: instance_v_ptr,
				instance_t:     instance_t,
				instance_v:     instance_v,
				field_t:        field_t,
				field_v:        field_v,
			}
		}
	}

	return Define(instance, m)
}
