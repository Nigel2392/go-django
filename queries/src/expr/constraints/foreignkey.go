package constraints

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"

	"github.com/Nigel2392/go-django/queries/src/deconstruct"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/queries/src/expr/builder"
)

var _ deconstruct.Deconstructible = (*foreignKeyConstraint)(nil)

func init() {
	deconstruct.Constructors.Register("queries.expr.constraints.ForeignKey", func(args []jsontext.Value, kwargs deconstruct.Kwargs, opts ...json.Options) (any, error) {
		if len(args) != 3 {
			return nil, fmt.Errorf("expected 3 arg for ForeignKeyConstraint, got %d", len(args))
		}

		var fk = &foreignKeyConstraint{
			Identifier: kwargs.Get[string]("identifier"),
			Error:      kwargs.Get[string]("error"),
		}

		if err := json.Unmarshal(args[0], &fk.Field, opts...); err != nil {
			return fk, err
		}
		if err := json.Unmarshal(args[1], &fk.OnUpdate, opts...); err != nil {
			return fk, err
		}
		if err := json.Unmarshal(args[2], &fk.OnDelete, opts...); err != nil {
			return fk, err
		}

		return fk, nil
	})
}

type foreignKeyConstraint struct {
	Base
	Field    expr.Expression
	OnUpdate MutationAction
	OnDelete MutationAction
	used     bool
}

func ForeignKeyConstraint(id string, errorMessage string, field string, onUpdate, onDelete MutationAction) expr.Expression {
	return &foreignKeyConstraint{
		Identifier: id,
		Error:      errorMessage,
		Field:      expr.Field(field),
		OnUpdate:   onUpdate,
		OnDelete:   onDelete,
	}
}

func (e *foreignKeyConstraint) Deconstruct() (path string, args []any, kwargs map[string]any) {
	return "queries.expr.constraints.ForeignKey", []any{e.Field, e.OnUpdate, e.OnDelete}, map[string]any{"identifier": e.Identifier, "error": e.Error}
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
