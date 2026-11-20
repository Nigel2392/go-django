package constraints

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"

	"github.com/Nigel2392/go-django/queries/src/deconstruct"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/queries/src/expr/builder"
)

var _ deconstruct.Deconstructible = (*checkConstraint)(nil)

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

func (e *checkConstraint) Deconstruct() (path string, args []any, kwargs map[string]any) {
	return "queries.expr.constraints.Check", []any{e.Statement}, map[string]any{"identifier": e.Identifier, "error": e.Error}
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
