package htmlutil

import (
	"strings"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/assert"
)

func AppendAttrs[T ~map[string]interface{}](attrs T, key string, value any) {
	var values []string
	switch v := value.(type) {
	case string:
		values = []string{v}
	case []string:
		values = v
	default:
		assert.Fail(errors.TypeMismatch.Wrapf(
			"expected attribute %s to be string or []string, got %T",
			key, value,
		))
	}

	existing, ok := attrs[key]
	if !ok {
		attrs[key] = strings.Join(values, " ")
		return
	}

	switch e := existing.(type) {
	case string:
		attrs[key] = strings.Join(append(values, e), " ")
	case []string:
		attrs[key] = strings.Join(append(values, e...), " ")
	default:
		assert.Fail(errors.TypeMismatch.Wrapf(
			"expected attribute %s to be string or []string, got %T",
			key, existing,
		))
	}
}
