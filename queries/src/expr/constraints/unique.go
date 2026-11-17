package constraints

import (
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/queries/src/expr/builder"
)

type uniqueConstraint struct {
	Base
	Fields []expr.Expression
	used   bool
}

func UniqueConstraint(id string, errorMessage string, fields any) expr.Expression {
	return &uniqueConstraint{
		Identifier: id,
		Error:      errorMessage,
		Fields:     expr.ExpressionFromInterface[expr.Expression](fields, false),
	}
}

func (e *uniqueConstraint) Clone() expr.Expression {
	var clonedFields = make([]expr.Expression, 0, len(e.Fields))
	for _, field := range e.Fields {
		clonedFields = append(clonedFields, field.Clone())
	}

	return &uniqueConstraint{
		Base:   e.Base,
		Fields: clonedFields,
	}
}

func (e *uniqueConstraint) Resolve(inf *expr.ExpressionInfo) expr.Expression {
	if e.used {
		return e
	}

	var nE = e.Clone().(*uniqueConstraint)
	nE.used = true

	for i, field := range nE.Fields {
		nE.Fields[i] = field.Resolve(inf)
	}

	return nE
}

func (e *uniqueConstraint) SQL(sb builder.Builder) {
}
