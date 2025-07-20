package attrs

import (
	"reflect"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/forms/fields"
)

type definerField struct {
	fields.Field
	attrField Field
}

func newDefinerField(f Field, opts ...func(fields.Field)) fields.Field {
	var df = &definerField{
		Field:     fields.CharField(opts...),
		attrField: f,
	}
	return df
}

func (f *definerField) ValueToForm(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	if IsZero(value) {
		return nil
	}

	switch v := value.(type) {
	case Definer:
		var defs = v.FieldDefs()
		var prim = defs.Primary()
		return prim.GetValue()
	default:
		return value
	}
}

func (f *definerField) ValueToGo(value interface{}) (interface{}, error) {
	var rT = f.attrField.Type()
	if rT.Kind() == reflect.Ptr {
		rT = rT.Elem()
	}

	var rV = reflect.New(rT)
	if definer, ok := rV.Interface().(Definer); ok {
		var defs = definer.FieldDefs()
		var prim = defs.Primary()
		var err = prim.Scan(value)
		if err != nil {
			return nil, err
		}
		return rV.Interface(), nil
	}

	return nil, errors.TypeMismatch.Wrapf(
		"Value %v (%T) is not a Definer",
		value, value,
	)
}
