package fields

import (
	"github.com/Nigel2392/go-django/src/internal/forms"
)

type (
	Field      = forms.Field
	FormWidget = forms.Widget
)

type SaveableField interface {
	Field
	Save(value interface{}) (interface{}, error)
}
