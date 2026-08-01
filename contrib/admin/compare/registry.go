package compare

import (
	"context"
	"reflect"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/trans"
)

type comparisonRegistry struct {
	// the default factory to use if no type or kind match is found
	// if nil, an error will be returned when no match is found
	// if set, this factory will be used as a fallback
	defaultFactory ComparisonFactory

	// a map of reflect.type to comparison initializer functions
	// this will be checked before the kind registry
	registry map[reflect.Type]ComparisonFactory

	// a map of reflect.kind to comparison initializer functions
	// this will be checked if no exact type match is found
	comparisonKindRegistry map[reflect.Kind]ComparisonFactory
}

var reg = &comparisonRegistry{
	defaultFactory:         nil,
	registry:               make(map[reflect.Type]ComparisonFactory),
	comparisonKindRegistry: make(map[reflect.Kind]ComparisonFactory),
}

func GetComparison(ctx context.Context, typ any, label any, fieldname string, oldInstance, newInstance attrs.Definer) (Comparison, error) {
	var (
		t reflect.Type
		k reflect.Kind
	)

	switch v := typ.(type) {
	case reflect.Type:
		t = v
		k = t.Kind()
	case reflect.Kind:
		t = nil
		k = v
	case nil:
		// the type can be retrieved from the meta field
		// this is handled below
	default:
		panic(errors.TypeMismatch.Wrapf(
			"provided type must be reflect.Type or reflect.Kind, got %T",
			typ,
		))
	}

	var meta attrs.ModelMeta
	if oldInstance != nil {
		meta = attrs.GetModelMeta(oldInstance)
	} else if newInstance != nil {
		meta = attrs.GetModelMeta(newInstance)
	}

	var labelFn = trans.GetTextFunc(label)
	if labelFn == nil {
		var field, ok = meta.Definitions().Field(fieldname)
		if !ok {
			return nil, errors.FieldNotFound.Wrapf(
				"field %q not found in model %T",
				fieldname, meta.Model(),
			)
		}
		labelFn = field.Label
	}

	// if no type was provided, try to get it from the meta field
	if t == nil {
		var field, ok = meta.Definitions().Field(fieldname)
		if !ok {
			return nil, errors.FieldNotFound.Wrapf(
				"field %q not found in model %T",
				fieldname, meta.Model(),
			)
		}
		t = field.Type()
		k = t.Kind()
	}

	if t != nil {
		if factory, ok := reg.registry[t]; ok {
			return factory(ctx, labelFn, fieldname, meta, oldInstance, newInstance)
		}
	}

	if factory, ok := reg.comparisonKindRegistry[k]; ok {
		return factory(ctx, labelFn, fieldname, meta, oldInstance, newInstance)
	}

	if reg.defaultFactory != nil {
		return reg.defaultFactory(ctx, labelFn, fieldname, meta, oldInstance, newInstance)
	}

	return nil, errors.NotImplemented.Wrapf(
		"no comparison registered for type %v (kind %v)", t, k,
	)
}

func RegisterComparisonType(typ reflect.Type, factory ComparisonFactory) {
	reg.registry[typ] = factory
}

func RegisterComparisonKind(kind reflect.Kind, factory ComparisonFactory) {
	reg.comparisonKindRegistry[kind] = factory
}

func RegisterDefaultComparison(factory ComparisonFactory) (replaced bool) {
	replaced = reg.defaultFactory != nil
	reg.defaultFactory = factory
	return replaced
}
