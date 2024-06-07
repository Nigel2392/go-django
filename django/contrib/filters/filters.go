package filters

import (
	"net/http"

	"github.com/Nigel2392/django/forms"
)

type FilterForm[T any] interface {
	forms.FormRenderer
	IsValid() bool
	Filter([]T) []T
}

type Filter[T any] interface {
	Form(prefix string, r *http.Request, initial map[string]interface{}) FilterForm[T]
}
