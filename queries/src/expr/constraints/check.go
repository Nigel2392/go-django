package constraints

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"

	"github.com/Nigel2392/go-django/queries/src/deconstruct"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/queries/src/expr/builder"
)

var (
	_ deconstruct.Deconstructible = (*checkConstraint)(nil)
	_ Constraint                  = (*checkConstraint)(nil)
)

func init() {
	deconstruct.Constructors.Register("queries.expr.constraints.Check", func(args []jsontext.Value, kwargs deconstruct.Kwargs, opts ...json.Options) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("expected 1 arg for CheckConstraint, got %d", len(args))
		}
		var cc = &checkConstraint{
			Base: Base{
				Identifier: kwargs.Get[string]("identifier"),
				Error:      kwargs.Get[string]("error"),
			},
		}
		return cc, json.Unmarshal(args[0], &cc.Statement, opts...)
	})
}

type checkConstraint struct {
	Base
	Statement expr.Expression
	used      bool
}

func (e *checkConstraint) Type() ConstraintType {
	return C_CHECK
}

func (e *checkConstraint) Deconstruct() (path string, args []any, kwargs map[string]any) {
	return "queries.expr.constraints.Check", []any{e.Statement}, map[string]any{"identifier": e.Identifier, "error": e.Error}
}

func CheckConstraint(id string, errorMessage string, statement expr.Expression) Constraint {
	return &checkConstraint{
		Base: Base{
			Identifier: id,
			Error:      errorMessage,
		},
		Statement: statement,
	}
}

func (e *checkConstraint) Clone() expr.Expression {
	var stmt expr.Expression
	if e.Statement != nil {
		stmt = e.Statement.Clone()
	}
	return &checkConstraint{
		Base:      e.Base,
		Statement: stmt,
		used:      e.used,
	}
}

func (e *checkConstraint) Resolve(inf *expr.ExpressionInfo) expr.Expression {
	if e.used {
		return e
	}

	var nE = e.Clone().(*checkConstraint)
	nE.used = true
	if inf == nil {
		return nE
	}

	cpy := *inf
	cpy.ForUpdate = true
	cpy.SupportsWhereAlias = false
	cpy.SupportsAsExpr = false
	nE.inf = &cpy

	if nE.Statement != nil {
		nE.Statement = nE.Statement.Resolve(nE.inf)
	}
	return nE
}

func (e *checkConstraint) SQL(sb builder.Builder) {
	if !e.used || e.inf == nil {
		sb.AddError(fmt.Errorf("check constraint %q must be resolved before generating SQL", e.Identifier))
		return
	}
	if e.Statement == nil {
		sb.AddError(fmt.Errorf("check constraint %q has nil statement", e.Identifier))
		return
	}
	if e.Identifier != "" && e.inf.QuoteIdentifier == nil {
		sb.AddError(fmt.Errorf("check constraint %q requires ExpressionInfo.QuoteIdentifier", e.Identifier))
		return
	}

	if e.Identifier != "" {
		sb.WriteString("CONSTRAINT ")
		sb.WriteString(e.inf.QuoteIdentifier(e.Identifier))
		sb.WriteString(" ")
	}
	sb.WriteString("CHECK (")
	e.Statement.SQL(sb)
	sb.WriteString(")")
}
