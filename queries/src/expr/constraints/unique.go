package constraints

import (
	"encoding/json/jsontext"
	"encoding/json/v2"

	"github.com/Nigel2392/go-django/queries/src/deconstruct"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/queries/src/expr/builder"
)

var _ deconstruct.Deconstructible = (*uniqueConstraint)(nil)

func init() {
	deconstruct.Constructors.Register("queries.expr.constraints.Unique", func(args []jsontext.Value, kwargs deconstruct.Kwargs, opts ...json.Options) (any, error) {
		var fields = make([]expr.Expression, len(args))
		for i, arg := range args {
			if err := json.Unmarshal(arg, &fields[i], opts...); err != nil {
				return nil, err
			}
		}
		return &uniqueConstraint{
			Base: Base{
				Identifier: kwargs.Get[string]("identifier"),
				Error:      kwargs.Get[string]("error"),
			},
			Fields: fields,
		}, nil
	})
}

type uniqueConstraint struct {
	Base
	Fields []expr.Expression
	used   bool
}

func (e *uniqueConstraint) Deconstruct() (path string, args []any, kwargs map[string]any) {
	var exprArgs = make([]any, len(e.Fields))
	for i, field := range e.Fields {
		exprArgs[i] = field
	}
	return "queries.expr.constraints.Unique", exprArgs, map[string]any{"identifier": e.Identifier, "error": e.Error}
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
