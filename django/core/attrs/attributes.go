package attrs

import "fmt"

func SetMany(d Definer, values map[string]interface{}) {
	for name, value := range values {
		Set(d, name, value)
	}
}

func Set(d Definer, name string, value interface{}) {
	set(d, name, value, false)
}

func ForceSet(d Definer, name string, value interface{}) {
	set(d, name, value, true)
}

func Get[T any](d Definer, name string) T {
	var defs = d.FieldDefs()
	var f, ok = defs.Field(name)
	if !ok {
		panic(
			fmt.Sprintf("get (%T): no field named %q", d, name),
		)
	}

	var v = f.GetValue()
	switch t := v.(type) {
	case T:
		return t
	case *T:
		return *t
	default:
		panic(
			fmt.Sprintf("get (%T): field %q is not of type %T", d, name, v),
		)
	}
}

func set(d Definer, name string, value interface{}, force bool) {
	var defs = d.FieldDefs()
	var f, ok = defs.Field(name)
	if !ok {
		panic(
			fmt.Sprintf("set (%T): no field named %q", d, name),
		)
	}

	f.SetValue(value, force)
}
