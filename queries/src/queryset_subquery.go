package queries

import (
	"context"
	"strings"

	_ "unsafe"

	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

//go:linkname newFunc github.com/Nigel2392/go-django/queries/src/expr.newFunc
func newFunc(funcLookup string, value []any, expr ...any) *expr.Function

var _ expr.Expression = (*subqueryExpr[attrs.Definer, *QuerySet[attrs.Definer]])(nil)

type subqueryExpr[T attrs.Definer, QS BaseQuerySet[T, QS]] struct {
	field expr.Expression
	qs    QS
	op    string
	not   bool
	used  bool
}

func (s *subqueryExpr[T, QS]) SQL(sb *strings.Builder) []any {
	var written bool
	var args = make([]any, 0)
	if s.field != nil {
		args = append(args, s.field.SQL(sb)...)
		written = true
	}

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
	var sql = query.SQL()
	if sql != "" {
		if written {
			sb.WriteString(" ")
		}

		sb.WriteString(sql)
	}

	args = append(args, query.Args()...)
	return args
}

func (s *subqueryExpr[T, QS]) Clone() expr.Expression {
	return &subqueryExpr[T, QS]{
		qs:    s.qs.Clone(),
		not:   s.not,
		used:  s.used,
		field: s.field,
		op:    s.op,
	}
}

func (s *subqueryExpr[T, QS]) Resolve(inf *expr.ExpressionInfo) expr.Expression {
	if inf.Model == nil || s.used {
		return s
	}

	var nE = s.Clone().(*subqueryExpr[T, QS])
	nE.used = true

	if nE.field != nil {
		nE.field = nE.field.Resolve(inf)
	}

	return nE
}

type subqueryContextKey struct{}

func IsSubqueryContext(ctx context.Context) bool {
	var v, ok = ctx.Value(subqueryContextKey{}).(bool)
	if !ok {
		return false
	}
	return v
}

func makeSubqueryContext(ctx context.Context) context.Context {
	if IsSubqueryContext(ctx) {
		return ctx
	}
	return context.WithValue(ctx, subqueryContextKey{}, true)
}

func Subquery[T attrs.Definer, QS BaseQuerySet[T, QS]](qs QS) expr.Expression {
	return &subqueryExpr[T, QS]{
		qs: qs.Limit(0).WithContext(makeSubqueryContext(qs.Context())),
	}
}
