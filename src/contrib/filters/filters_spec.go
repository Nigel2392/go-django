package filters

import (
	"errors"

	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/forms/fields"
)

type BaseFilterSpec[ListT any] struct {
	SpecName  string
	FormField fields.Field
	Apply     func(value interface{}, objectList []ListT) error
}

func (b *BaseFilterSpec[ListT]) Name() string {
	return b.SpecName
}

func (b *BaseFilterSpec[ListT]) Field() fields.Field {
	return b.FormField
}

func (b *BaseFilterSpec[ListT]) Filter(value interface{}, objectList []ListT) error {
	if b.Apply == nil {
		logger.Fatalf(1, "Apply function is not defined for filter %s", b.Name())
		return errors.New("apply function is not defined")
	}
	return b.Apply(value, objectList)
}
