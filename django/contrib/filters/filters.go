package filters

import (
	"net/http"

	"github.com/Nigel2392/django/forms"
)

type FilterForm[T any] interface {
	forms.FormRenderer
	IsValid() bool
	EntryFilter([]T) []T
}

type EntryFilter[T any] interface {
	Form(prefix string, r *http.Request, initial map[string]interface{}) FilterForm[T]
}
