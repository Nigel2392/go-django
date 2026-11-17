package constraints

import (
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/queries/src/expr/builder"
)

type checkConstraint struct {
	Base
	Statement expr.Expression
	used      bool
}

func CheckConstraint(id string, errorMessage string, field string, onUpdate, onDelete MutationAction) expr.Expression {
	return &checkConstraint{
		Identifier: id,
		Error:      errorMessage,
		Statement:  expr.Field(field),
	}
}

func (e *checkConstraint) Clone() expr.Expression {
	return &checkConstraint{
		Base:      e.Base,
		Statement: e.Statement.Clone(),
	}
}

func (e *checkConstraint) Resolve(inf *expr.ExpressionInfo) expr.Expression {
	if e.used {
		return e
	}

	var nE = e.Clone().(*checkConstraint)
	nE.used = true
	nE.Statement = nE.Statement.Resolve(inf)
	return nE
}

func (e *checkConstraint) SQL(sb builder.Builder) {
}
