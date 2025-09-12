package compare

import (
	"context"
	"html/template"

	"github.com/Nigel2392/go-django/src/core/attrs"
)

type Comparison interface {
	Label(ctx context.Context) string
	HTMLDiff(ctx context.Context) (template.HTML, error)
	HasChanged(ctx context.Context) (bool, error)
}

type ComparisonWrapper interface {
	Comparison
	Unwrap() []Comparison
}

type ComparisonFactory func(ctx context.Context, fieldname string, old, new attrs.Definer) (Comparison, error)
