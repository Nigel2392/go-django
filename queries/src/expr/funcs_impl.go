package expr

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"slices"

	"github.com/Nigel2392/go-django/queries/src/deconstruct"
	"github.com/Nigel2392/go-django/queries/src/expr/builder"
)

func init() {
	deconstruct.Constructors.Register("queries.expr.Function", func(args []jsontext.Value, kwargs deconstruct.Kwargs, opts ...json.Options) (any, error) {
		funcLookup, ok := kwargs.GetOk[string]("func")
		if !ok || funcLookup == "" {
			return nil, fmt.Errorf("missing func name for Function")
		}
		var inner = make([]Expression, len(args))
		for i, arg := range args {
			if err := json.Unmarshal(arg, &inner[i], opts...); err != nil {
				return nil, err
			}
		}
		var params []any
		if rawParams, hasParams := kwargs["params"]; hasParams && rawParams != nil {
			b, err := json.Marshal(rawParams, opts...)
			if err != nil {
				return nil, err
			}
			v, err := unmarshalAnyArg(b, opts...)
			if err != nil {
				return nil, err
			}
			paramSlice, ok := v.([]any)
			if !ok {
				return nil, fmt.Errorf("invalid params type %T for Function", v)
			}
			params = paramSlice
		}
		return &Function{
			funcLookup: funcLookup,
			fieldName:  kwargs.Get[string]("field"),
			args:       params,
			inner:      inner,
		}, nil
	})
}

var _ deconstruct.Deconstructible = (*Function)(nil)

type Function struct {
	sql        func(col []Expression, funcParams []any) (sql string, args []any, err error)
	funcLookup string
	fieldName  string
	args       []any
	used       bool
	inner      []Expression
}

func (e *Function) Deconstruct() (string, []any, map[string]any) {
	var args = make([]any, 0, len(e.inner))
	for _, inner := range e.inner {
		args = append(args, inner)
	}
	var kwargs = map[string]any{
		"func": e.funcLookup,
	}
	if e.fieldName != "" {
		kwargs["field"] = e.fieldName
	}
	if len(e.args) > 0 {
		kwargs["params"] = slices.Clone(e.args)
	}
	return "queries.expr.Function", args, kwargs
}

func (e *Function) FieldName() string {
	if e.fieldName != "" {
		return e.fieldName
	}

	for _, expr := range e.inner {
		if namer, ok := expr.(NamedExpression); ok {
			var name = namer.FieldName()
			if name != "" {
				return name
			}
		}
	}

	return ""
}

func (e *Function) SQL(sb builder.Builder) {
	if e.sql == nil {
		panic(fmt.Errorf("SQL function %v not provided", e.funcLookup))
	}

	sql, params, err := e.sql(
		slices.Clone(e.inner),
		slices.Clone(e.args),
	)

	if err != nil {
		sb.AddError(err)
		return
	}

	sb.WriteString(sql)
	sb.AddVar(params...)
}

func (e *Function) Clone() Expression {
	var inner = slices.Clone(e.inner)
	for i := range inner {
		inner[i] = inner[i].Clone()
	}

	return &Function{
		sql:        e.sql,
		funcLookup: e.funcLookup,
		fieldName:  e.fieldName,
		args:       slices.Clone(e.args),
		used:       e.used,
		inner:      inner,
	}
}

func (e *Function) Resolve(inf *ExpressionInfo) Expression {
	if e.used {
		return e
	}

	var nE = e.Clone().(*Function)
	nE.used = true

	if len(nE.inner) > 0 {
		for i, inner := range nE.inner {
			nE.inner[i] = inner.Resolve(inf)
		}
	}

	var sql, ok = funcLookups.lookupFunc(
		inf.Driver, nE.funcLookup,
	)
	if !ok {
		panic(fmt.Errorf("could not resolve SQL function %q", nE.funcLookup))
	}

	nE.sql = func(col []Expression, funcParams []any) (string, []any, error) {
		return sql(inf, col, funcParams)
	}

	return nE
}

type LogicalNamedExpressionFunc interface {
	NamedExpression
	LogicalExpression
}

func newFunc(funcLookup string, value []any, expr ...any) LogicalNamedExpressionFunc {
	var inner = make([]Expression, 0, len(expr))
	for _, e := range expr {
		switch v := e.(type) {
		case ExpressionBuilder:
			inner = append(inner, v.BuildExpression())
		case Expression:
			inner = append(inner, v)
		case string:
			inner = append(inner, Field(v))
		default:
			panic("unsupported type")
		}
	}

	return newChainExpr(&Function{
		funcLookup: funcLookup,
		args:       value,
		inner:      inner,
	})
}

func SUM(expr ...any) LogicalNamedExpressionFunc {
	return newFunc("SUM", []any{}, expr...)
}

func COUNT(expr ...any) LogicalNamedExpressionFunc {
	return newFunc("COUNT", []any{}, expr...)
}

func AVG(expr ...any) LogicalNamedExpressionFunc {
	return newFunc("AVG", []any{}, expr...)
}

func MAX(expr ...any) LogicalNamedExpressionFunc {
	return newFunc("MAX", []any{}, expr...)
}

func MIN(expr ...any) LogicalNamedExpressionFunc {
	return newFunc("MIN", []any{}, expr...)
}

func COALESCE(expr ...any) LogicalNamedExpressionFunc {
	return newFunc("COALESCE", []any{}, expr...)
}

func CONCAT(expr ...any) LogicalNamedExpressionFunc {
	return newFunc("CONCAT", []any{}, expr...)
}

func SUBSTR(expr any, start, length any) LogicalNamedExpressionFunc {
	return newFunc("SUBSTR", []any{start, length}, expr)
}

func EXISTS(expr any) LogicalNamedExpressionFunc {
	return newFunc("EXISTS", []any{}, expr)
}

func UPPER(expr any) LogicalNamedExpressionFunc {
	return newFunc("UPPER", []any{}, expr)
}

func LOWER(expr any) LogicalNamedExpressionFunc {
	return newFunc("LOWER", []any{}, expr)
}

func LENGTH(expr any) LogicalNamedExpressionFunc {
	return newFunc("LENGTH", []any{}, expr)
}

func NOW() LogicalNamedExpressionFunc {
	return newFunc("NOW", []any{})
}

func UTCNOW() LogicalNamedExpressionFunc {
	return newFunc("UTCNOW", []any{})
}

func LOCALTIMESTAMP() LogicalNamedExpressionFunc {
	return newFunc("LOCALTIMESTAMP", []any{})
}

func DATE(expr any) LogicalNamedExpressionFunc {
	return newFunc("DATE", []any{}, expr)
}

func DATE_FORMAT(expr any, format string) LogicalNamedExpressionFunc {
	return newFunc("DATE_FORMAT", []any{format}, expr)
}
