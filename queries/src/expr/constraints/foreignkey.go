package constraints

import (
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/queries/src/expr/builder"
)

type foreignKeyConstraint struct {
	Base
	Field expr.Expression
	used  bool
}

func ForeignKeyConstraint(id string, errorMessage string, field string, onUpdate, onDelete MutationAction) expr.Expression {
	return &foreignKeyConstraint{
		Identifier: id,
		Error:      errorMessage,
		Field:      expr.Field(field),
	}
}

func (e *foreignKeyConstraint) Clone() expr.Expression {
	return &foreignKeyConstraint{
		Base:  e.Base,
		Field: e.Field.Clone(),
	}
}

func (e *foreignKeyConstraint) Resolve(inf *expr.ExpressionInfo) expr.Expression {
	if e.used {
		return e
	}

	var nE = e.Clone().(*foreignKeyConstraint)
	nE.used = true
	nE.Field = nE.Field.Resolve(inf)
	return nE
}

func (e *foreignKeyConstraint) SQL(sb builder.Builder) {
}
