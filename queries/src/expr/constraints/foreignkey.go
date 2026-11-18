package constraints

import (
	"encoding/json/jsontext"
	"encoding/json/v2"

	"github.com/Nigel2392/go-django/queries/src/deconstruct"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/queries/src/expr/builder"
)

var _ deconstruct.Deconstructible = (*foreignKeyConstraint)(nil)

func init() {
	deconstruct.Constructors.Register("queries.expr.constraints.ForeignKey", func(args []jsontext.Value, kwargs deconstruct.Kwargs, opts ...json.Options) (any, error) {
		var fk = &foreignKeyConstraint{
			Identifier: kwargs.Get[string]("identifier"),
			Error:      kwargs.Get[string]("error"),
		}

		return fk, json.Unmarshal(args[0], &fk.Field, opts...)
	})
}

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

func (e *foreignKeyConstraint) Deconstruct() (path string, args []any, kwargs map[string]any) {
	return "queries.expr.constraints.ForeignKey", []any{e.Field}, map[string]any{"identifier": e.Identifier, "error": e.Error}
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
