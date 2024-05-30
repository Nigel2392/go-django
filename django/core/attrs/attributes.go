package attrs

import (
	"fmt"

	"github.com/Nigel2392/django/core/assert"
)

func SetMany(d Definer, values map[string]interface{}) error {
	for name, value := range values {
		if err := assert.Err(set(d, name, value, false)); err != nil {
			return err
		}
	}
	return nil
}

func Set(d Definer, name string, value interface{}) error {
	return set(d, name, value, false)
}

func ForceSet(d Definer, name string, value interface{}) error {
	return set(d, name, value, true)
}

func Get[T any](d Definer, name string) T {
	var defs = d.FieldDefs()
	var f, ok = defs.Field(name)
	if !ok {
		assert.Fail(
			"get (%T): no field named %q",
			d, name,
		)
	}

	var v = f.GetValue()
	switch t := v.(type) {
	case T:
		return t
	case *T:
		return *t
	default:
		assert.Fail(
			"get (%T): field %q is not of type %T",
			d, name, v,
		)
	}
	return *(new(T))
}

func set(d Definer, name string, value interface{}, force bool) error {
	var defs = d.FieldDefs()
	var f, ok = defs.Field(name)
	if !ok {
		return assert.Fail(
			fmt.Sprintf("set (%T): no field named %q", d, name),
		)
	}

	return f.SetValue(value, force)
}
