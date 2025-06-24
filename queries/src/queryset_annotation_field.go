package queries

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	"strings"

	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/forms/fields"
)

var (
	_ VirtualField = &queryField[any]{}
	_ VirtualField = &exprField{}
)

// a field used internally in queryset annotations
// to represent a query expression.
type queryField[T any] struct {
	name  string
	expr  expr.Expression
	value T
}

func newQueryField[T any](name string, expr expr.Expression) *queryField[T] {
	return &queryField[T]{name: name, expr: expr}
}

func (q *queryField[T]) BindToDefinitions(defs attrs.Definitions) {
	// this is a no-op, as query fields are not bound to definitions
	// they are used directly in expressions and queries, not anywhere external
	// or as part of a model definition.
}

func (q *queryField[T]) FieldDefinitions() attrs.Definitions {
	// query fields do not have field definitions, as they are not part of a model
	// they are used directly in expressions and queries, not anywhere external
	// or as part of a model definition.
	return nil
}

// VirtualField

func (q *queryField[T]) Alias() string { return q.name }
func (q *queryField[T]) SQL(inf *expr.ExpressionInfo) (string, []any) {
	var sqlBuilder = &strings.Builder{}
	var expr = q.expr.Resolve(inf)
	var args = expr.SQL(sqlBuilder)
	return sqlBuilder.String(), args
}

// attrs.Field minimal impl
func (q *queryField[T]) Name() string          { return q.name }
func (q *queryField[T]) ColumnName() string    { return "" }
func (q *queryField[T]) Tag(string) string     { return "" }
func (q *queryField[T]) Type() reflect.Type    { return reflect.TypeOf(*new(T)) }
func (q *queryField[T]) Attrs() map[string]any { return map[string]any{} }
func (q *queryField[T]) IsPrimary() bool       { return false }
func (q *queryField[T]) AllowNull() bool       { return true }
func (q *queryField[T]) AllowBlank() bool      { return true }
func (q *queryField[T]) AllowEdit() bool       { return false }
func (q *queryField[T]) GetValue() any         { return q.value }
func (q *queryField[T]) SetValue(v any, _ bool) error {
	val, ok := v.(T)
	if !ok {
		return fmt.Errorf("type mismatch on queryField[%T]: %v", *new(T), v)
	}
	q.value = val
	return nil
}
func (q *queryField[T]) Value() (driver.Value, error) { return q.value, nil }
func (q *queryField[T]) Scan(v any) error             { return q.SetValue(v, false) }
func (q *queryField[T]) GetDefault() any              { return nil }
func (q *queryField[T]) Instance() attrs.Definer      { return nil }
func (q *queryField[T]) Rel() attrs.Relation          { return nil }
func (q *queryField[T]) FormField() fields.Field      { return nil }
func (q *queryField[T]) Validate() error              { return nil }
func (q *queryField[T]) Label() string                { return q.name }
func (q *queryField[T]) ToString() string             { return fmt.Sprint(q.value) }
func (q *queryField[T]) HelpText() string             { return "" }

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
