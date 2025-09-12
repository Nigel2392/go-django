package compare

import (
	"context"
	"reflect"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

type comparisonRegistry struct {
	// a map of reflect.type to comparison initializer functions
	// this will be checked before the kind registry
	registry map[reflect.Type]ComparisonFactory

	// a map of reflect.kind to comparison initializer functions
	// this will be checked if no exact type match is found
	comparisonKindRegistry map[reflect.Kind]ComparisonFactory
}

var reg = &comparisonRegistry{
	registry:               make(map[reflect.Type]ComparisonFactory),
	comparisonKindRegistry: make(map[reflect.Kind]ComparisonFactory),
}

func GetComparison(ctx context.Context, typ any, fieldname string, old, new attrs.Definer) (Comparison, error) {
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
		panic(errors.TypeMismatch.Wrap(
			"provided type cannot be nil",
		))
	default:
		panic(errors.TypeMismatch.Wrapf(
			"provided type must be reflect.Type or reflect.Kind, got %T",
			typ,
		))
	}

	if t != nil {
		if factory, ok := reg.registry[t]; ok {
			return factory(ctx, fieldname, old, new)
		}
	}

	if factory, ok := reg.comparisonKindRegistry[k]; ok {
		return factory(ctx, fieldname, old, new)
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
