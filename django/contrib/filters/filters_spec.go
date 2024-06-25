package filters

import "github.com/Nigel2392/django/forms/fields"

type BaseFilterSpec[ListT any] struct {
	SpecName  string
	FormField fields.Field
	Apply     func(value interface{}, objectList []ListT)
}

func (b *BaseFilterSpec[ListT]) Name() string {
	return b.SpecName
}

func (b *BaseFilterSpec[ListT]) Field() fields.Field {
	return b.FormField
}

func (b *BaseFilterSpec[ListT]) Filter(value interface{}, objectList []ListT) {
	b.Apply(value, objectList)
}
