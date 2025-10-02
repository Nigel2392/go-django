package fields

import (
	"reflect"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

type EmbedOptions struct {
	// AutoInit indicates whether the pointer to the model should be automatically initialized
	// If AutoInit is true, the model will be initialized to a new instance
	// If it is false, the model will be left as nil and no fields will be embedded.
	AutoInit bool

	// EmbedFields specifies which fields of the model should be embedded
	// If EmbedFields is nil, all fields of the model will be embedded.
	EmbedFields []any
}

type embedder struct {
	fieldName any
	options   EmbedOptions
}

func (e *embedder) bind(d attrs.Definer) []attrs.Field {
	var (
		rTyp = reflect.TypeOf(d).Elem()
		rVal = reflect.ValueOf(d).Elem()
	)

	var fieldval reflect.Value
	switch v := e.fieldName.(type) {
	case string:
		var fieldTyp, ok = rTyp.FieldByName(v)
		assert.True(ok, "field %q not found in %T", v, d)

		fieldval = rVal.FieldByIndex(fieldTyp.Index)
		assert.True(
			fieldval.Kind() == reflect.Ptr && fieldval.Type().Elem().Kind() == reflect.Struct,
			"field %q in %T must be a pointer to a struct, got %s", v, d, fieldval.Kind(),
		)
		assert.True(
			fieldval.CanSet(),
			"field %q in %T must be settable, got %s", v, d, fieldval.Kind(),
		)
	case attrs.Definer:
		fieldval = reflect.ValueOf(v)
		e.fieldName = reflect.TypeOf(v).Elem().Name()
	default:
		assert.Fail("nameOrScan must be a string or attrs.Definer, got %T", v)
	}

	if fieldval.IsNil() {
		if e.options.AutoInit {
			var newVal = reflect.ValueOf(attrs.NewObject[attrs.Definer](
				fieldval.Type().Elem(),
			))
			fieldval.Set(newVal)
		} else {
			return []attrs.Field{} // no fields to embed
		}
	}

	definer, ok := fieldval.Interface().(attrs.Definer)
	assert.True(ok, "field %q in %T must implement attrs.Definer, got %T", e.fieldName, d, fieldval.Interface())

	if len(e.options.EmbedFields) > 0 {
		var fields, err = attrs.UnpackFieldsFromArgs(definer, e.options.EmbedFields)
		assert.True(err == nil, "failed to unpack fields: %v", err)
		return fields
	}

	return definer.FieldDefs().Fields()
}

func Embed(nameOrScan any, options ...EmbedOptions) func(d attrs.Definer) []attrs.Field {
	var opts EmbedOptions
	if len(options) > 0 {
		opts = options[0]
	}

	var e = &embedder{
		fieldName: nameOrScan,
		options:   opts,
	}

	return e.bind
}
