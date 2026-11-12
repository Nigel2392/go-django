package fields

import (
	"context"
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
	ctx       context.Context
}

func (e *embedder) bind(d attrs.Definer) []attrs.Field {
	var (
		rTyp = reflect.TypeOf(d).Elem()
		rVal = reflect.ValueOf(d).Elem()
	)

	var (
		fieldval  reflect.Value
		fieldName = e.fieldName
	)
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
		fieldName = reflect.TypeOf(v).Elem().Name()
	default:
		rv := reflect.ValueOf(v)
		if rv.IsValid() && rv.Kind() == reflect.Ptr && rv.Type().Elem().Implements(reflect.TypeFor[attrs.Definer]()) {
			fieldval = rv.Elem()
			fieldName = fieldval.Type().Elem().Name()
			break
		}
		assert.Fail("nameOrScan must be a string or attrs.Definer, got %T", v)
	}

	if fieldval.IsNil() {
		if e.options.AutoInit {
			var newVal = reflect.ValueOf(attrs.NewObject[attrs.Definer](
				e.ctx, fieldval.Type().Elem(),
			))
			fieldval.Set(newVal)
		} else {
			return []attrs.Field{} // no fields to embed
		}
	}

	definer, ok := fieldval.Interface().(attrs.Definer)
	assert.True(ok, "field %q in %T must implement attrs.Definer, got %T", fieldName, d, fieldval.Interface())

	if len(e.options.EmbedFields) > 0 {
		var (
			defs attrs.Definitions
			args = make([]any, len(e.options.EmbedFields))
		)
		for i, item := range e.options.EmbedFields {
			if str, ok := item.(string); ok {
				if defs == nil {
					defs = attrs.Define(e.ctx, definer)
				}
				if f, ok := defs.Field(str); ok {
					args[i] = f
					continue
				}
			}
			args[i] = item
		}
		var fields, err = attrs.UnpackFieldsFromArgs(e.ctx, definer, args...)
		assert.True(err == nil, "failed to unpack fields: %v", err)
		return fields
	}

	return definer.FieldDefs(e.ctx).Fields()
}

func Embed(ctx context.Context, nameOrScan any, options ...EmbedOptions) func(d attrs.Definer) []attrs.Field {
	var opts EmbedOptions
	if len(options) > 0 {
		opts = options[0]
	}

	ctx = attrs.ContextWithFlags(
		ctx, attrs.CtxFlagEmbedded, true,
	)

	var e = &embedder{
		fieldName: nameOrScan,
		options:   opts,
		ctx:       ctx,
	}

	return e.bind
}
