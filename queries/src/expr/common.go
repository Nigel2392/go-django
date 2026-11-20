package expr

import (
	"database/sql/driver"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"reflect"
	"slices"
	"time"

	"github.com/Nigel2392/go-django/queries/src/deconstruct"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/expr/builder"
)

func init() {
	deconstruct.Constructors.Register("queries.expr.String", func(args []jsontext.Value, kwargs deconstruct.Kwargs, opts ...json.Options) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("expected 1 arg for String, got %d", len(args))
		}
		var s string
		if err := json.Unmarshal(args[0], &s, opts...); err != nil {
			return nil, err
		}
		return String(s), nil
	})
	deconstruct.Constructors.Register("queries.expr.Field", func(args []jsontext.Value, kwargs deconstruct.Kwargs, opts ...json.Options) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("expected 1 arg for Field, got %d", len(args))
		}
		var f field
		if err := json.Unmarshal(args[0], &f.fieldName, opts...); err != nil {
			return nil, err
		}
		return &f, nil
	})
	deconstruct.Constructors.Register("queries.expr.Values", func(args []jsontext.Value, kwargs deconstruct.Kwargs, opts ...json.Options) (any, error) {
		var vals = make([]any, len(args))
		for i, arg := range args {
			v, err := unmarshalAnyArg(arg, opts...)
			if err != nil {
				return nil, err
			}
			vals[i] = v
		}
		return &values{values: vals}, nil
	})
	deconstruct.Constructors.Register("queries.expr.Value", func(args []jsontext.Value, kwargs deconstruct.Kwargs, opts ...json.Options) (any, error) {
		if len(args) > 1 {
			return nil, fmt.Errorf("expected at most 1 arg for Value, got %d", len(args))
		}
		var val any
		if len(args) == 1 {
			var err error
			val, err = unmarshalAnyArg(args[0], opts...)
			if err != nil {
				return nil, err
			}
		} else if rawVal, ok := kwargs["value"]; ok && rawVal != nil {
			b, err := json.Marshal(rawVal, opts...)
			if err != nil {
				return nil, err
			}
			val, err = unmarshalAnyArg(b, opts...)
			if err != nil {
				return nil, err
			}
		}
		return &value{v: val, unsafe: kwargs.Get[bool]("unsafe")}, nil
	})
	deconstruct.Constructors.Register("queries.expr.As", func(args []jsontext.Value, kwargs deconstruct.Kwargs, opts ...json.Options) (any, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("expected 2 args for As, got %d", len(args))
		}
		var f namedExpression
		if err := json.Unmarshal(args[0], &f.fieldName, opts...); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(args[1], &f.Expression, opts...); err != nil {
			return nil, err
		}
		return &f, nil
	})
}

var (
	_ deconstruct.Deconstructible = String("")
	_ deconstruct.Deconstructible = (*field)(nil)
	_ deconstruct.Deconstructible = (*values)(nil)
	_ deconstruct.Deconstructible = (*value)(nil)
	_ deconstruct.Deconstructible = (*namedExpression)(nil)
)

// StringExpr is a string type which implements the Expression interface.
// It is used to represent a string value in SQL queries.
//
// It can be used like so, and supports no arguments:
//
//	StringExpr("a = b")
type String string

func (e String) Deconstruct() (string, []any, map[string]any) {
	return "queries.expr.String", []any{string(e)}, nil
}

func (e String) String() string                         { return string(e) }
func (e String) SQL(sb builder.Builder)                 { sb.WriteString(string(e)) }
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

func (e *field) Deconstruct() (string, []any, map[string]any) {
	return "queries.expr.Field", []any{e.fieldName}, nil
}

func (e *field) FieldName() string {
	return e.fieldName
}

func (e *field) SQL(sb builder.Builder) {
	sb.WriteString(e.field.SQLText)
	sb.AddVar(e.field.SQLArgs...)
}

func (e *field) Clone() Expression {
	return &field{fieldName: e.fieldName, field: e.field, used: e.used}
}

func (e *field) Resolve(inf *ExpressionInfo) Expression {
	if e.used {
		return e
	}

	var nE = e.Clone().(*field)
	nE.used = true
	nE.field = inf.ResolveExpressionField(nE.fieldName)
	return nE
}

// values is a struct that implements the Expression interface.
// It is used to represent multiple values in SQL queries.
//
// these values will always be wrapped in parentheses
// and can be used in IN clauses or other expressions that require multiple values.
type values struct {
	values      []any
	used        bool
	placeholder string        // Placeholder for the values, if needed
	driver      driver.Driver // Driver used for the expression, for type casting
}

// Values creates a values expression from a slice of values.
// It is used to represent multiple values in SQL queries, allowing for both safe and unsafe usage.
// It can be used like so:
//
//	Values([]any{"value1", "value2"})
func Values(vs ...any) Expression {
	if len(vs) == 0 {
		panic("values cannot be empty")
	}

	return &values{
		values: slices.Clone(vs),
		used:   false,
	}
}

func (e *values) Deconstruct() (string, []any, map[string]any) {
	return "queries.expr.Values", slices.Clone(e.values), nil
}

func (e *values) SQL(sb builder.Builder) {

	sb.WriteString("(")

	for i, v := range e.values {
		if i > 0 {
			sb.WriteString(", ")
		}

		sb.WriteString(e.placeholder)

		if _, ok := e.driver.(*drivers.DriverPostgres); ok {
			var rVal = reflect.ValueOf(v)
			switch rVal.Kind() {
			case reflect.Invalid:
				// nil literal requires no explicit type cast on Postgres
			case reflect.String:
				sb.WriteString("::TEXT")
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				sb.WriteString("::INT")
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
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
				switch v.(type) {
				case time.Time, *time.Time:
				default:
					sb.AddError(fmt.Errorf("unsupported value type %T in expression", v))
					return
				}
			}
		}
	}

	sb.WriteString(")")
	sb.AddVar(e.values...)
}

func (e *values) Clone() Expression {
	return &values{
		values:      slices.Clone(e.values),
		used:        e.used,
		placeholder: e.placeholder,
		driver:      e.driver,
	}
}

func (e *values) Resolve(inf *ExpressionInfo) Expression {
	if e.used {
		return e
	}

	var nE = e.Clone().(*values)
	nE.used = true
	nE.placeholder = inf.Placeholder
	nE.driver = inf.Driver
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

func (e *value) Deconstruct() (string, []any, map[string]any) {
	var kwargs map[string]any
	if e.unsafe {
		kwargs = map[string]any{"unsafe": true}
	}
	return "queries.expr.Value", []any{e.v}, kwargs
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

	if exprBuilder, ok := v.(ExpressionBuilder); ok {
		return exprBuilder.BuildExpression()
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

func (e *value) SQL(sb builder.Builder) {
	if e.unsafe {
		sb.WriteString(fmt.Sprintf("%v", e.v))
		return
	}

	sb.WriteString(e.placeholder)

	// Explicitly handle postgres type casting to ensure
	// the value is correctly interpreted by the database.
	// This is necessary because Postgres does not automatically cast
	// values to the correct type in all cases.
	if _, ok := e.driver.(*drivers.DriverPostgres); ok {
		var rVal = reflect.ValueOf(e.v)
		switch rVal.Kind() {
		case reflect.Invalid:
			// nil literal requires no explicit type cast on Postgres
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
			switch e.v.(type) {
			case time.Time, *time.Time:
			default:
				sb.AddError(fmt.Errorf("unsupported value type %T in expression", e.v))
				return
			}
		}
	}

	sb.AddVar(e.v)
}

func (e *value) Clone() Expression {
	return &value{v: e.v, used: e.used, unsafe: e.unsafe}
}

func (e *value) Resolve(inf *ExpressionInfo) Expression {
	if e.used {
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
	fmtRhs    func(op LogicalOp, rhs string, values ...any) (string, []any)
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

func (n *namedExpression) Deconstruct() (string, []any, map[string]any) {
	return "queries.expr.As", []any{n.fieldName, n.Expression}, nil
}

func (n *namedExpression) FieldName() string {
	return n.fieldName
}

func (n *namedExpression) Annotate() bool {
	return true
}

func (n *namedExpression) Clone() Expression {
	return &namedExpression{
		used:       n.used,
		field:      n.field,
		fieldName:  n.fieldName,
		forUpdate:  n.forUpdate,
		fmtRhs:     n.fmtRhs,
		Expression: n.Expression.Clone(),
	}
}

func (n *namedExpression) Resolve(inf *ExpressionInfo) Expression {
	if n.used {
		return n
	}

	var nE = n.Clone().(*namedExpression)
	nE.used = true
	nE.forUpdate = inf.ForUpdate
	nE.fmtRhs = inf.Lookups.FormatLogicalOpRHS

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

	/*
		set SupportsAsExpr to false in case [n] it gets used in queryset select
		[namedExpression] can be used to alias existing fields, it can be possible to:

		```
			queryset.Select(expr.As("ID", "ID"))
		```

		this could be useful for union querysets.
		the queryset will then add an annotation (see [namedExpression.Annotate])
		but there will be an issue: the expression will reference the newly added
		annotations' alias setting SupportsAsExpr will make sure that any expression
		that gets resolved will output their full SQL, instead of self-referencing nothing.
	*/
	cpy.SupportsAsExpr = false
	nE.Expression = nE.Expression.Resolve(&cpy)
	return nE
}

func (n *namedExpression) SQL(sb builder.Builder) {
	if !n.forUpdate {
		n.Expression.SQL(sb)
		return
	}

	var exprSb builder.BaseBuilder
	n.Expression.SQL(&exprSb)
	var opRHS, _ = n.fmtRhs(
		EQ, exprSb.String(),
	)

	sb.WriteString(n.field.SQLText)
	sb.WriteString(opRHS)
	sb.AddVar(exprSb.Vars...)
	sb.AddError(exprSb.Errors...)
}
