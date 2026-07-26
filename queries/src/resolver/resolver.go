package resolver

import (
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

// OrderByDefiner is an interface that can be implemented by models to indicate
// that the model has a default ordering that should be used when executing queries.
type OrderByDefiner interface {
	attrs.Definer
	OrderBy() []string
}

type ExpressionCompiler interface {
	// ExpressionInfo returns a usable [expr.ExpressionInfo],
	// allowing for the use of GO field names in a raw SQL query.
	//
	// This is used (for example) to parse raw queries inside of [QuerySet.Rows], [QuerySet.Row] and [QuerySet.Exec].
	ExpressionInfo(resolver expr.FieldResolver) *expr.ExpressionInfo
}
