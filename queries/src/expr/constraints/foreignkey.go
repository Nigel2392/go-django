package constraints

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"

	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/queries/src/expr/builder"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/deconstruct"
)

var (
	_ deconstruct.Deconstructible = (*foreignKeyConstraint)(nil)
	_ Constraint                  = (*foreignKeyConstraint)(nil)
)

func init() {
	deconstruct.Constructors.Register("queries.expr.constraints.ForeignKey", func(args []jsontext.Value, kwargs deconstruct.Kwargs, opts ...json.Options) (any, error) {
		if len(args) != 3 {
			return nil, fmt.Errorf("expected 3 arg for ForeignKeyConstraint, got %d", len(args))
		}

		var fk = &foreignKeyConstraint{
			Base: Base{
				Identifier: kwargs.Get[string]("identifier"),
				Error:      kwargs.Get[string]("error"),
			},
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
	Field        expr.Expression
	OnUpdate     MutationAction
	OnDelete     MutationAction
	used         bool
	targetTable  string
	targetColumn string
	resolveErr   error
}

func (e *foreignKeyConstraint) Type() ConstraintType {
	return C_FOREIGN_KEY
}

func (e *foreignKeyConstraint) ForeignKeyField() string {
	if named, ok := e.Field.(expr.NamedExpression); ok {
		return named.FieldName()
	}
	return ""
}

func ForeignKeyConstraint(id string, errorMessage string, field string, onUpdate, onDelete MutationAction) Constraint {
	return &foreignKeyConstraint{
		Base: Base{
			Identifier: id,
			Error:      errorMessage,
		},
		Field:    expr.Field(field),
		OnUpdate: onUpdate,
		OnDelete: onDelete,
	}
}

func (e *foreignKeyConstraint) Deconstruct() (path string, args []any, kwargs map[string]any) {
	return "queries.expr.constraints.ForeignKey", []any{e.Field, e.OnUpdate, e.OnDelete}, map[string]any{"identifier": e.Identifier, "error": e.Error}
}

func (e *foreignKeyConstraint) Clone() expr.Expression {
	var fld expr.Expression
	if e.Field != nil {
		fld = e.Field.Clone()
	}
	return &foreignKeyConstraint{
		Base:         e.Base,
		Field:        fld,
		OnUpdate:     e.OnUpdate,
		OnDelete:     e.OnDelete,
		used:         e.used,
		targetTable:  e.targetTable,
		targetColumn: e.targetColumn,
		resolveErr:   e.resolveErr,
	}
}

func (e *foreignKeyConstraint) Resolve(inf *expr.ExpressionInfo) expr.Expression {
	if e.used {
		return e
	}

	var nE = e.Clone().(*foreignKeyConstraint)
	nE.used = true
	if inf == nil || inf.Resolver == nil {
		nE.resolveErr = fmt.Errorf("foreign key constraint %q requires ExpressionInfo with Resolver", e.Identifier)
		return nE
	}

	cpy := *inf
	cpy.ForUpdate = true
	cpy.SupportsWhereAlias = false
	cpy.SupportsAsExpr = false
	nE.inf = &cpy

	if nE.Field == nil {
		nE.resolveErr = fmt.Errorf("foreign key constraint %q has nil field", e.Identifier)
		return nE
	}

	named, ok := nE.Field.(expr.NamedExpression)
	if !ok {
		nE.resolveErr = fmt.Errorf("foreign key constraint %q field %T must implement NamedExpression", e.Identifier, nE.Field)
		return nE
	}

	fieldName := named.FieldName()
	_, fieldDef, _, err := inf.Resolver.Resolve(fieldName, inf)
	if err != nil {
		nE.resolveErr = fmt.Errorf("foreign key constraint %q: resolve field %q: %w", e.Identifier, fieldName, err)
		return nE
	}

	rel := fieldDef.Rel()
	if rel == nil || rel.Model() == nil {
		nE.resolveErr = fmt.Errorf("foreign key constraint %q: field %q has no relation", e.Identifier, fieldName)
		return nE
	}

	targetDefs := attrs.GetModelMeta(rel.Model()).Definitions()
	targetTable := targetDefs.TableName()
	if targetTable == "" {
		nE.resolveErr = fmt.Errorf("foreign key constraint %q: related model %T has no table name", e.Identifier, rel.Model())
		return nE
	}

	targetField := rel.Field()
	if targetField == nil {
		targetField = targetDefs.Primary()
	}
	if targetField == nil {
		nE.resolveErr = fmt.Errorf("foreign key constraint %q: related model %T has no target/primary field", e.Identifier, rel.Model())
		return nE
	}

	targetColumn := targetField.ColumnName()
	if targetColumn == "" {
		nE.resolveErr = fmt.Errorf("foreign key constraint %q: related field %q has empty column name", e.Identifier, targetField.Name())
		return nE
	}

	nE.targetTable = targetTable
	nE.targetColumn = targetColumn
	nE.Field = nE.Field.Resolve(nE.inf)
	return nE
}

func (e *foreignKeyConstraint) SQL(sb builder.Builder) {
	if !e.used || e.inf == nil {
		if e.resolveErr != nil {
			sb.AddError(e.resolveErr)
			return
		}
		sb.AddError(fmt.Errorf("foreign key constraint %q must be resolved before generating SQL", e.Identifier))
		return
	}
	if e.resolveErr != nil {
		sb.AddError(e.resolveErr)
		return
	}
	if e.Field == nil {
		sb.AddError(fmt.Errorf("foreign key constraint %q has nil field", e.Identifier))
		return
	}
	if e.inf.QuoteIdentifier == nil {
		sb.AddError(fmt.Errorf("foreign key constraint %q requires ExpressionInfo.QuoteIdentifier", e.Identifier))
		return
	}

	if e.Identifier != "" {
		sb.WriteString("CONSTRAINT ")
		sb.WriteString(e.inf.QuoteIdentifier(e.Identifier))
		sb.WriteString(" ")
	}
	sb.WriteString("FOREIGN KEY (")
	e.Field.SQL(sb)
	sb.WriteString(") REFERENCES ")
	sb.WriteString(e.inf.QuoteIdentifier(e.targetTable))
	sb.WriteString(" (")
	sb.WriteString(e.inf.QuoteIdentifier(e.targetColumn))
	sb.WriteString(") ON DELETE ")
	sb.WriteString(e.OnDelete.String())
	sb.WriteString(" ON UPDATE ")
	sb.WriteString(e.OnUpdate.String())
}
