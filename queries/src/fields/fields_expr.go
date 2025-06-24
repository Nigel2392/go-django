package fields

import (
	"strings"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

var _ queries.VirtualField = (*ExpressionField[any])(nil)

type ExpressionField[T any] struct {
	*DataModelField[T]

	// expr is the expression used to calculate the field's value
	expr expr.Expression
}

func NewVirtualField[T any](forModel attrs.Definer, dst any, name string, expr expr.Expression) *ExpressionField[T] {
	var f = &ExpressionField[T]{
		DataModelField: NewDataModelField[T](forModel, dst, name),
		expr:           expr,
	}
	f.DataModelField.fieldRef = f // Set the field reference to itself
	f.DataModelField.setupInitialVal()
	return f
}

func (f *ExpressionField[T]) Alias() string {
	return f.DataModelField.Name()
}

func (f *ExpressionField[T]) SQL(inf *expr.ExpressionInfo) (string, []any) {
	if f.expr == nil {
		return "", nil
	}
	var expr = f.expr.Resolve(inf)
	var sb strings.Builder
	var args = expr.SQL(&sb)
	return sb.String(), args
}
