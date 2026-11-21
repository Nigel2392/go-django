package constraints

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"

	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/queries/src/expr/builder"
	"github.com/Nigel2392/go-django/src/core/deconstruct"
)

var (
	_ deconstruct.Deconstructible = (*uniqueConstraint)(nil)
	_ Constraint                  = (*uniqueConstraint)(nil)
)

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

func (e *uniqueConstraint) Type() ConstraintType {
	return C_UNIQUE
}

func (e *uniqueConstraint) Deconstruct() (path string, args []any, kwargs map[string]any) {
	var exprArgs = make([]any, len(e.Fields))
	for i, field := range e.Fields {
		exprArgs[i] = field
	}
	return "queries.expr.constraints.Unique", exprArgs, map[string]any{"identifier": e.Identifier, "error": e.Error}
}

func UniqueConstraint(id string, errorMessage string, fields ...any) Constraint {
	return &uniqueConstraint{
		Base: Base{
			Identifier: id,
			Error:      errorMessage,
		},
		Fields: expr.ExpressionFromInterface[expr.Expression](fields, false),
	}
}

func (e *uniqueConstraint) Clone() expr.Expression {
	var clonedFields = make([]expr.Expression, 0, len(e.Fields))
	for _, field := range e.Fields {
		if field == nil {
			clonedFields = append(clonedFields, nil)
			continue
		}
		clonedFields = append(clonedFields, field.Clone())
	}

	return &uniqueConstraint{
		Base:   e.Base,
		Fields: clonedFields,
		used:   e.used,
	}
}

func (e *uniqueConstraint) Resolve(inf *expr.ExpressionInfo) expr.Expression {
	if e.used {
		return e
	}

	var nE = e.Clone().(*uniqueConstraint)
	nE.used = true
	if inf == nil {
		return nE
	}

	cpy := *inf
	cpy.ForUpdate = true
	cpy.SupportsWhereAlias = false
	cpy.SupportsAsExpr = false
	nE.inf = &cpy

	for i, field := range nE.Fields {
		if field != nil {
			nE.Fields[i] = field.Resolve(nE.inf)
		}
	}

	return nE
}

func (e *uniqueConstraint) SQL(sb builder.Builder) {
	if !e.used || e.inf == nil {
		sb.AddError(fmt.Errorf("unique constraint %q must be resolved before generating SQL", e.Identifier))
		return
	}
	if len(e.Fields) == 0 {
		sb.AddError(fmt.Errorf("unique constraint %q has no fields", e.Identifier))
		return
	}
	for i, field := range e.Fields {
		if field == nil {
			sb.AddError(fmt.Errorf("unique constraint %q has nil field at index %d", e.Identifier, i))
			return
		}
	}
	if e.Identifier != "" && e.inf.QuoteIdentifier == nil {
		sb.AddError(fmt.Errorf("unique constraint %q requires ExpressionInfo.QuoteIdentifier", e.Identifier))
		return
	}

	if e.Identifier != "" {
		sb.WriteString("CONSTRAINT ")
		sb.WriteString(e.inf.QuoteIdentifier(e.Identifier))
		sb.WriteString(" ")
	}
	sb.WriteString("UNIQUE (")
	for i, field := range e.Fields {
		if i > 0 {
			sb.WriteString(", ")
		}
		field.SQL(sb)
	}
	sb.WriteString(")")
}
