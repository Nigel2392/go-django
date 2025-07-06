package queries

import (
	"strings"

	_ "unsafe"

	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

//go:linkname newFunc github.com/Nigel2392/go-django/queries/src/expr.newFunc
func newFunc(funcLookup string, value []any, expr ...any) *expr.Function

var _ expr.Expression = (*subqueryExpr)(nil)

type subqueryExpr struct {
	field expr.Expression
	q     QueryInfo
	op    string
	not   bool
	used  bool
}

func (s *subqueryExpr) SQL(sb *strings.Builder) []any {
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

	var sql = s.q.SQL()
	if sql != "" {
		if written {
			sb.WriteString(" ")
		}

		sb.WriteString(sql)
	}

	args = append(args, s.q.Args()...)
	return args
}

func (s *subqueryExpr) Clone() expr.Expression {
	return &subqueryExpr{
		q:     s.q,
		not:   s.not,
		used:  s.used,
		field: s.field,
		op:    s.op,
	}
}

func (s *subqueryExpr) Resolve(inf *expr.ExpressionInfo) expr.Expression {
	if inf.Model == nil || s.used {
		return s
	}

	var nE = s.Clone().(*subqueryExpr)
	nE.used = true

	if nE.field != nil {
		nE.field = nE.field.Resolve(inf)
	}

	return nE
}

func Subquery[T attrs.Definer, QS BaseQuerySet[T, QS]](qs QS) expr.Expression {
	q := qs.Limit(0).QueryAll()
	return &subqueryExpr{
		q: q,
	}
}
