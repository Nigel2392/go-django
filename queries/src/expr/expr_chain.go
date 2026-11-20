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
	deconstruct.Constructors.Register("queries.expr.Chain", func(args []jsontext.Value, kwargs deconstruct.Kwargs, opts ...json.Options) (any, error) {
		if len(args) == 0 {
			return nil, fmt.Errorf("expected at least 1 arg for Chain, got 0")
		}
		var inner = make([]Expression, len(args))
		for i, arg := range args {
			if err := json.Unmarshal(arg, &inner[i], opts...); err != nil {
				return nil, err
			}
		}
		return &chainExpr{
			inner: inner,
		}, nil
	})
}

var _ deconstruct.Deconstructible = (*chainExpr)(nil)

type chainExpr struct {
	used      bool
	forUpdate bool
	inner     []Expression
}

func (e *chainExpr) Deconstruct() (string, []any, map[string]any) {
	var args = make([]any, len(e.inner))
	for i, inner := range e.inner {
		args[i] = inner
	}
	return "queries.expr.Chain", args, nil
}

func (e *chainExpr) FieldName() string {
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

func (e *chainExpr) SQL(sb builder.Builder) {
	if len(e.inner) == 0 {
		sb.AddError(fmt.Errorf("SQL chainExpr has no inner expressions"))
		return
	}

	for _, inner := range e.inner {
		inner.SQL(sb)
	}
}

func (e *chainExpr) Clone() Expression {
	var inner = slices.Clone(e.inner)
	for i := range inner {
		inner[i] = inner[i].Clone()
	}

	return &chainExpr{
		used:      e.used,
		forUpdate: e.forUpdate,
		inner:     inner,
	}
}

func (e *chainExpr) Resolve(inf *ExpressionInfo) Expression {
	if e.used {
		return e
	}

	var nE = e.Clone().(*chainExpr)

	nE.used = true
	nE.forUpdate = inf.ForUpdate

	if len(nE.inner) > 0 {
		for i, inner := range nE.inner {
			nE.inner[i] = inner.Resolve(inf)
		}
	}

	return nE
}

func Chain(expr ...any) NamedExpression {
	var inner = make([]Expression, 0, len(expr))
	var fieldName string

	for i, e := range expr {

		if exprBuilder, ok := e.(ExpressionBuilder); ok {
			e = exprBuilder.BuildExpression()
		}

		if n, ok := e.(NamedExpression); ok && (i == 0 || i > 0 && fieldName == "") {
			fieldName = n.FieldName()
		}

		switch v := e.(type) {
		case LogicalOp:
			inner = append(inner, v)
		case Expression:
			inner = append(inner, v)
		case string:
			inner = append(inner, Field(v))
		default:
			panic("unsupported type")
		}
	}

	if len(inner) == 0 {
		panic(fmt.Errorf("chainExpr requires at least one inner expression"))
	}

	return &chainExpr{
		inner: inner,
	}
}
