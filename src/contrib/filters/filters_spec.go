package filters

import (
	"errors"
	"net/http"

	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/forms/fields"
)

type BaseFilterSpec[T any] struct {
	SpecName  string
	FormField fields.Field
	Apply     func(req *http.Request, value interface{}, object T) (T, error)
}

func (b *BaseFilterSpec[T]) Name() string {
	return b.SpecName
}

func (b *BaseFilterSpec[T]) Field(r *http.Request) fields.Field {
	return b.FormField
}

func (b *BaseFilterSpec[T]) Filter(req *http.Request, value interface{}, object T) (T, error) {
	if b.Apply == nil {
		logger.Fatalf(1, "Apply function is not defined for filter %s", b.Name())
		return *new(T), errors.New("apply function is not defined")
	}
	return b.Apply(req, value, object)
}
