package attrs

import (
	"fmt"
	"reflect"
	"strings"
)

func autoDefinitionStructTag(t reflect.StructField) (blank bool, editable bool) {
	var (
		tag   = t.Tag.Get("attrs")
		split = strings.Split(tag, "|")
	)
	blank, editable = false, true

	for _, s := range split {
		switch strings.TrimSpace(s) {
		case "blank":
			blank = true
		case "readonly":
			editable = false
		}
	}
	return blank, editable
}

func AutoDefinitions[T Definer](instance T, include ...string) Definitions {
	var m = make(map[string]Field)

	var (
		instance_t_ptr = reflect.TypeOf(instance)
		instance_v_ptr = reflect.ValueOf(instance)
	)

	if instance_t_ptr.Kind() != reflect.Ptr {
		panic(
			fmt.Sprintf("AutoDefinitions: %T is not a pointer", instance),
		)
	}

	var (
		instance_t = instance_t_ptr.Elem()
		instance_v = instance_v_ptr.Elem()
	)

	if len(include) == 0 {
		for i := 0; i < instance_t.NumField(); i++ {
			var (
				field_t         = instance_t.Field(i)
				field_v         = instance_v.Field(i)
				name            = field_t.Name
				blank, editable = autoDefinitionStructTag(field_t)
			)

			var skip = (field_t.Anonymous ||
				field_t.PkgPath != "" ||
				field_t.Tag.Get("attrs") == "-")

			if skip {
				continue
			}

			m[name] = &FieldDef{
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
			if !ok {
				panic(
					fmt.Sprintf("AutoDefinitions: field %q not found in %T", name, instance),
				)
			}
			var (
				blank, editable = autoDefinitionStructTag(field_t)
				field_v         = instance_v.FieldByIndex(field_t.Index)
			)

			var skip = (field_t.Anonymous ||
				field_t.PkgPath != "" ||
				field_t.Tag.Get("attrs") == "-")

			if skip {
				continue
			}

			m[name] = &FieldDef{
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
