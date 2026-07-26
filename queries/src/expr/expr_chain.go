package expr

import (
	"fmt"
	"slices"

	"github.com/Nigel2392/go-django/queries/src/expr/builder"
)

type chainExpr struct {
	used      bool
	forUpdate bool
	inner     []Expression
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
