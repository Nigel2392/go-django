package expr

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	"strings"

	"github.com/Nigel2392/go-django/queries/src/drivers"
)

// StringExpr is a string type which implements the Expression interface.
// It is used to represent a string value in SQL queries.
//
// It can be used like so, and supports no arguments:
//
//	StringExpr("a = b")
type String string

func (e String) String() string                         { return string(e) }
func (e String) SQL(sb *strings.Builder) []any          { sb.WriteString(string(e)); return []any{} }
func (e String) Clone() Expression                      { return String([]byte(e)) }
func (e String) Resolve(inf *ExpressionInfo) Expression { return e }

// field is a string type which implements the Expression interface.
// It is used to represent a field in SQL queries.
// It can be used like so:
//
//	Field("MyModel.MyField")
type field struct {
	fieldName string
	field     *ResolvedField
	used      bool
}

func Field(fld string) NamedExpression {
	return &field{fieldName: fld}
}

func (e *field) FieldName() string {
	return e.fieldName
}

func (e *field) SQL(sb *strings.Builder) []any {
	sb.WriteString(e.field.SQLText)
	return e.field.SQLArgs
}

func (e *field) Clone() Expression {
	return &field{fieldName: e.fieldName, field: e.field, used: e.used}
}

func (e *field) Resolve(inf *ExpressionInfo) Expression {
	if inf.Model == nil || e.used {
		return e
	}

	var nE = e.Clone().(*field)
	nE.used = true
	nE.field = inf.ResolveExpressionField(nE.fieldName)
	return nE
}

// value is a type that implements the Expression interface.
// See [Value] for more information.
type value struct {
	v           any
	used        bool
	unsafe      bool
	driver      driver.Driver
	placeholder string // Placeholder for the value, if needed
}

// Value is a function that creates a value expression.
// It is used to represent a value in SQL queries, allowing for both safe and unsafe usage.
// It can be used like so:
//
//	Value("some value") // safe usage
//	Value("some value", true) // unsafe usage, will not use placeholders
//
// The unsafe usage allows for direct insertion of values into the SQL query, which can be dangerous if not used carefully.
func Value(v any, unsafe ...bool) Expression {
	if expr, ok := v.(Expression); ok {
		return expr
	}

	var s bool
	if len(unsafe) > 0 && unsafe[0] {
		s = true
	}
	return &value{v: normalizeDefinerArg(v), unsafe: s}
}

// V is a shorthand for Value, allowing for a more concise syntax.
// See [Value] for more information.
func V(v any, unsafe ...bool) Expression {
	return Value(v, unsafe...)
}

func (e *value) SQL(sb *strings.Builder) []any {
	if e.unsafe {
		sb.WriteString(fmt.Sprintf("%v", e.v))
		return []any{}
	}

	sb.WriteString(e.placeholder)

	// Explicitly handle postgres type casting to ensure
	// the value is correctly interpreted by the database.
	// This is necessary because Postgres does not automatically cast
	// values to the correct type in all cases.
	if _, ok := e.driver.(*drivers.DriverPostgres); ok {
		var rVal = reflect.ValueOf(e.v)
		switch rVal.Kind() {
		case reflect.String:
			sb.WriteString("::TEXT")
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			sb.WriteString("::INT")
		case reflect.Float32, reflect.Float64:
			sb.WriteString("::FLOAT")
		case reflect.Bool:
			sb.WriteString("::BOOLEAN")
		case reflect.Slice, reflect.Array:
			if rVal.Type().Elem().Kind() == reflect.Uint8 {
				sb.WriteString("::BYTEA")
			} else {
				sb.WriteString("::TEXT[]")
			}
		default:
			panic(fmt.Errorf("unsupported value type %T in expression", e.v))
		}
	}

	return []any{e.v}
}

func (e *value) Clone() Expression {
	return &value{v: e.v, used: e.used, unsafe: e.unsafe}
}

func (e *value) Resolve(inf *ExpressionInfo) Expression {
	if inf.Model == nil || e.used {
		return e
	}

	var nE = e.Clone().(*value)
	nE.used = true
	nE.placeholder = inf.Placeholder
	nE.driver = inf.Driver

	if !nE.unsafe {
		return nE
	}

	switch v := any(nE.v).(type) {
	case string:
		nE.v = any(inf.Quote(v))
	case []byte:
		panic("cannot use []byte as a value in an expression, use a string instead")
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		nE.v = any(fmt.Sprintf("%d", v))
	case float32, float64:
		nE.v = any(fmt.Sprintf("%f", v))
	case bool:
		if v {
			nE.v = any("1")
		} else {
			nE.v = any("0")
		}
	case nil:
		nE.v = any("NULL")
	default:
		panic(fmt.Errorf("unsupported value type %T in expression", v))
	}

	return nE
}

type namedExpression struct {
	field     *ResolvedField
	fieldName string
	forUpdate bool
	used      bool
	Expression
}

// As creates a NamedExpression with a specified field name and an expression.
//
// It is used to give a name to an expression, which can be useful for annotations,
// or for updating fields in a model using [Expression]s.
func As(name string, expr Expression) NamedExpression {
	if name == "" {
		panic("field name cannot be empty")
	}
	if expr == nil {
		panic("expression cannot be nil")
	}
	return &namedExpression{
		fieldName:  name,
		Expression: expr,
	}
}

func (n *namedExpression) FieldName() string {
	return n.fieldName
}

func (n *namedExpression) Clone() Expression {
	return &namedExpression{
		used:       n.used,
		field:      n.field,
		fieldName:  n.fieldName,
		forUpdate:  n.forUpdate,
		Expression: n.Expression.Clone(),
	}
}

func (n *namedExpression) Resolve(inf *ExpressionInfo) Expression {
	if inf.Model == nil || n.used {
		return n
	}

	var nE = n.Clone().(*namedExpression)
	nE.used = true
	nE.forUpdate = inf.ForUpdate

	if nE.fieldName != "" && nE.forUpdate {
		nE.field = inf.ResolveExpressionField(nE.fieldName)

		if len(nE.field.SQLArgs) > 0 {
			panic(fmt.Errorf(
				"field %q cannot use arguments in an update expression, got %d (%v)",
				nE.fieldName, len(nE.field.SQLArgs), nE.field.SQLArgs,
			))
		}
	}

	// Dereference the info to copy it,
	// subexpressions should not handle the ForUpdate flag anymore,
	// as it is already handled by the namedExpression itself.
	var cpy = *inf
	cpy.ForUpdate = false
	nE.Expression = nE.Expression.Resolve(&cpy)
	return nE
}

func (n *namedExpression) SQL(sb *strings.Builder) []any {
	if !n.forUpdate {
		return n.Expression.SQL(sb)
	}

	sb.WriteString(n.field.SQLText)
	sb.WriteString(" = ")
	return n.Expression.SQL(sb)
}
