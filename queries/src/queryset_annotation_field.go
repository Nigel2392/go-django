package queries

import (
	"context"
	"database/sql/driver"
	"fmt"
	"reflect"
	"strings"

	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/forms/fields"
)

var (
	_ VirtualField = &queryField{}
	_ VirtualField = &exprField{}
)

// a field used internally in queryset annotations
// to represent a query expression.
type queryField struct {
	name  string
	expr  expr.Expression
	value any
}

func newQueryField(name string, expr expr.Expression) *queryField {
	return &queryField{name: name, expr: expr}
}

func (q *queryField) BindToDefinitions(defs attrs.Definitions) {
	// this is a no-op, as query fields are not bound to definitions
	// they are used directly in expressions and queries, not anywhere external
	// or as part of a model definition.
}

func (q *queryField) FieldDefinitions() attrs.Definitions {
	// query fields do not have field definitions, as they are not part of a model
	// they are used directly in expressions and queries, not anywhere external
	// or as part of a model definition.
	return nil
}

// VirtualField

func (q *queryField) Alias() string { return q.name }
func (q *queryField) SQL(inf *expr.ExpressionInfo) (string, []any) {
	var sqlBuilder = &strings.Builder{}
	var expr = q.expr.Resolve(inf)
	var args = expr.SQL(sqlBuilder)
	return sqlBuilder.String(), args
}

func (q *queryField) Expression() expr.Expression {
	return q.expr
}

// attrs.Field minimal impl
func (q *queryField) Name() string          { return q.name }
func (q *queryField) ColumnName() string    { return "" }
func (q *queryField) Tag(string) string     { return "" }
func (q *queryField) Type() reflect.Type    { return reflect.TypeOf(new(interface{})).Elem() }
func (q *queryField) Attrs() map[string]any { return map[string]any{} }
func (q *queryField) IsPrimary() bool       { return false }
func (q *queryField) AllowNull() bool       { return true }
func (q *queryField) AllowBlank() bool      { return true }
func (q *queryField) AllowEdit() bool       { return false }
func (q *queryField) GetValue() any         { return q.value }
func (q *queryField) SetValue(v any, _ bool) error {
	//	val, ok := v.(T)
	//	if !ok {
	//		return errors.TypeMismatch.WithCause(fmt.Errorf(
	//			"expected value of type %T, got %T: %v",
	//			*new(T), v, v,
	//		))
	//	}
	q.value = v
	return nil
}
func (q *queryField) Value() (driver.Value, error)        { return q.value, nil }
func (q *queryField) Scan(v any) error                    { return q.SetValue(v, false) }
func (q *queryField) GetDefault() any                     { return nil }
func (q *queryField) Instance() attrs.Definer             { return nil }
func (q *queryField) Rel() attrs.Relation                 { return nil }
func (q *queryField) FormField() fields.Field             { return nil }
func (q *queryField) Validate() error                     { return nil }
func (q *queryField) Label(ctx context.Context) string    { return q.name }
func (q *queryField) ToString() string                    { return fmt.Sprint(q.value) }
func (q *queryField) HelpText(ctx context.Context) string { return "" }

var _ VirtualField = &exprField{}

// a field used internally in queryset expressions
// to represent an expression that can be used in SQL queries.
//
// it wraps the provided [attrs.Field] and implements the [VirtualField] interface.
type exprField struct {
	attrs.Field
	expr expr.Expression
}

func (e *exprField) SQL(inf *expr.ExpressionInfo) (string, []any) {
	var sqlBuilder = &strings.Builder{}
	var expr = e.expr.Resolve(inf)
	var args = expr.SQL(sqlBuilder)
	return sqlBuilder.String(), args
}
