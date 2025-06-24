package queries

import (
	"strings"

	_ "unsafe"

	"github.com/Nigel2392/go-django/queries/src/expr"
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

		sb.WriteString("(")
		sb.WriteString(sql)
		sb.WriteString(")")
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

func Subquery(qs *GenericQuerySet) expr.Expression {
	if qs.internals.Limit == MAX_DEFAULT_RESULTS {
		qs.internals.Limit = 0
	}

	q := qs.queryAll()
	return &subqueryExpr{
		q: q,
	}
}

func SubqueryCount(qs *GenericQuerySet) *subqueryExpr {
	q := qs.queryCount()
	return &subqueryExpr{
		q:  q,
		op: "COUNT",
	}
}

func SubqueryExists(qs *GenericQuerySet) expr.Expression {
	q := qs.queryAll()
	return &subqueryExpr{
		q:  q,
		op: "EXISTS",
	}
}
