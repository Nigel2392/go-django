package compare

import (
	"context"
	"html/template"

	"github.com/Nigel2392/go-django/src/core/attrs"
)

type Comparison interface {
	Label() string
	HTMLDiff() (template.HTML, error)
	HasChanged() (bool, error)
}

type ComparisonWrapper interface {
	Comparison
	Unwrap() []Comparison
}

type ComparisonFactory func(ctx context.Context, label func(context.Context) string, fieldName string, modelMeta attrs.ModelMeta, old, new attrs.Definer) (Comparison, error)
