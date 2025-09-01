package tokens

import "github.com/Nigel2392/go-django/queries/src/drivers/errors"

type PartsProviderFunc[T any] func(obj T) ([]any, error)

func (f PartsProviderFunc[T]) TokenParts(obj any) ([]any, error) {
	return f(obj.(T))
}

type ifacePartsProvider struct{}

func (p *ifacePartsProvider) TokenParts(obj any) ([]any, error) {
	if _, ok := obj.(PartsProvider); ok {
		return obj.(PartsProvider).TokenParts()
	}
	return nil, errors.TypeMismatch.Wrapf(
		"object %T does not implement PartsProvider",
		obj,
	)
}
