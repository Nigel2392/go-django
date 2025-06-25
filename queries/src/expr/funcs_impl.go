package expr

import (
	"fmt"
	"slices"
	"strings"
)

type Function struct {
	sql        func(col []Expression, funcParams []any) (sql string, args []any, err error)
	funcLookup string
	fieldName  string
	args       []any
	used       bool
	inner      []Expression
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

func (e *Function) SQL(sb *strings.Builder) []any {
	if e.sql == nil {
		panic(fmt.Errorf("SQL function %v not provided", e.funcLookup))
	}

	sql, params, err := e.sql(
		slices.Clone(e.inner),
		slices.Clone(e.args),
	)

	if err != nil {
		panic(err)
	}

	sb.WriteString(sql)

	return params
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
	if inf.Model == nil || e.used {
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

func newFunc(funcLookup string, value []any, expr ...any) *Function {
	var inner = make([]Expression, 0, len(expr))
	for _, e := range expr {
		switch v := e.(type) {
		case Expression:
			inner = append(inner, v)
		case string:
			inner = append(inner, Field(v))
		default:
			panic("unsupported type")
		}
	}

	return &Function{
		funcLookup: funcLookup,
		args:       value,
		inner:      inner,
	}
}

func SUM(expr ...any) *Function {
	return newFunc("SUM", []any{}, expr...)
}

func COUNT(expr ...any) *Function {
	return newFunc("COUNT", []any{}, expr...)
}

func AVG(expr ...any) *Function {
	return newFunc("AVG", []any{}, expr...)
}

func MAX(expr ...any) *Function {
	return newFunc("MAX", []any{}, expr...)
}

func MIN(expr ...any) *Function {
	return newFunc("MIN", []any{}, expr...)
}

func COALESCE(expr ...any) *Function {
	return newFunc("COALESCE", []any{}, expr...)
}

func CONCAT(expr ...any) *Function {
	return newFunc("CONCAT", []any{}, expr...)
}

func SUBSTR(expr any, start, length any) *Function {
	return newFunc("SUBSTR", []any{start, length}, expr)
}

func UPPER(expr any) *Function {
	return newFunc("UPPER", []any{}, expr)
}

func LOWER(expr any) *Function {
	return newFunc("LOWER", []any{}, expr)
}

func LENGTH(expr any) *Function {
	return newFunc("LENGTH", []any{}, expr)
}

func NOW() *Function {
	return newFunc("NOW", []any{})
}

func UTCNOW() *Function {
	return newFunc("UTCNOW", []any{})
}

func LOCALTIMESTAMP() *Function {
	return newFunc("LOCALTIMESTAMP", []any{})
}

func DATE(expr any) *Function {
	return newFunc("DATE", []any{}, expr)
}

func DATE_FORMAT(expr any, format string) *Function {
	return newFunc("DATE_FORMAT", []any{format}, expr)
}
