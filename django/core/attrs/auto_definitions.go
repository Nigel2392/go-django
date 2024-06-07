package attrs

import (
	"reflect"
	"strconv"

	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/tags"
)

const ATTR_TAG_NAME = "attrs"

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
		case "primary":
			data.Primary = true
		case "label":
			data.Label = v[0]
		case "helpText":
			data.HelpText = v[0]
		case "default":
			var (
				default_ = v[0]
				val      any
				err      error
				chkTyp   = t.Type
			)

			if chkTyp.Kind() == reflect.Ptr {
				chkTyp = chkTyp.Elem()
			}

			switch chkTyp.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				val, err = strconv.ParseInt(default_, 10, 64)
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				val, err = strconv.ParseUint(default_, 10, 64)
			case reflect.Float32, reflect.Float64:
				val, err = strconv.ParseFloat(default_, 64)
			case reflect.String:
				val = default_
			case reflect.Bool:
				val, err = strconv.ParseBool(default_)
			default:
				assert.Fail("unsupported type %v", t.Type)
			}

			assert.True(err == nil, "error parsing %q: %v", default_, err)

			var r_v = reflect.ValueOf(val)
			assert.True(r_v.IsValid(), "invalid value %v", val)

			if r_v.Type() != t.Type && !r_v.Type().ConvertibleTo(t.Type) && t.Type.Kind() != reflect.Ptr {
				assert.Fail("type mismatch %v != %v", r_v.Type(), t.Type)
			}

			if t.Type.Kind() == reflect.Ptr && r_v.Type().Kind() != reflect.Ptr {
				// r_v = r_v.Addr()
				var new = reflect.New(t.Type.Elem())
				r_v = r_v.Convert(new.Elem().Type())
				new.Elem().Set(r_v)
				val = new.Interface()
			} else {
				val = r_v.Interface()
			}

			r_v = reflect.ValueOf(val)

			if r_v.Type() != t.Type {
				r_v = r_v.Convert(t.Type)
			}

			data.Default = r_v.Interface()
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
