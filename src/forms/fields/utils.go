package fields

import (
	"github.com/Nigel2392/go-django/src/internal/django_reflect"
)

// For compatibility purposes this function stays defined here.
// IsZero checks if the value is zero or not.
func IsZero(value interface{}) bool {
	return django_reflect.IsZero(value)
}
