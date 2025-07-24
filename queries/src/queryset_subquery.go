package queries

import (
	"strings"

	_ "unsafe"

	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

//go:linkname newFunc github.com/Nigel2392/go-django/queries/src/expr.newFunc
func newFunc(funcLookup string, value []any, expr ...any) *expr.Function

var _ expr.Expression = (*subqueryExpr[attrs.Definer, *QuerySet[attrs.Definer]])(nil)

type subqueryExpr[T attrs.Definer, QS BaseQuerySet[T, QS]] struct {
	qs   QS
	op   string
	not  bool
	used bool
}

func (s *subqueryExpr[T, QS]) SQL(sb *strings.Builder) []any {
	var written bool
	var args = make([]any, 0)

	if s.not {
		if written {
			sb.WriteString(" ")
		}

		sb.WriteString("NOT ")
		written = true
	}

	if s.op != "" {
		if written {
			sb.WriteString(" ")
		}

		sb.WriteString(s.op)
		written = true
	}

	var query = s.qs.QueryAll()
	var info = s.qs.Peek()
	var sql = query.SQL()
	if sql != "" {
		if written {
			sb.WriteString(" ")
		}

		if info.Limit == 1 {
			sb.WriteString("(")
		}

		sb.WriteString(sql)

		if info.Limit == 1 {
			sb.WriteString(")")
		}

		args = append(args, query.Args()...)
	}

	return args
}

func (s *subqueryExpr[T, QS]) Clone() expr.Expression {
	return &subqueryExpr[T, QS]{
		qs:   s.qs.Clone(),
		not:  s.not,
		used: s.used,
		op:   s.op,
	}
}

func (s *subqueryExpr[T, QS]) Resolve(inf *expr.ExpressionInfo) expr.Expression {
	if inf.Model == nil || s.used {
		return s
	}

	var nE = s.Clone().(*subqueryExpr[T, QS])
	nE.used = true

	// This allows for [expr.OuterRef] to work correctly
	nE.qs = nE.qs.WithContext(expr.AddParentSubqueryContext(
		nE.qs.Context(), inf,
	))

	return nE
}

func Subquery[T attrs.Definer, QS BaseQuerySet[T, QS]](qs QS) expr.Expression {
	return &subqueryExpr[T, QS]{
		qs: qs.WithContext(expr.MakeSubqueryContext(qs.Context())),
	}
}
