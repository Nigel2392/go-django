package expr

import (
	"fmt"
	"strings"
)

type when struct {
	lhs  Expression
	rhs  Expression
	then Expression
}

func When(keyOrExpr interface{}, vals ...any) *when {
	switch v := keyOrExpr.(type) {
	case string:
		if len(vals) == 0 {
			panic("no values provided for when clause (rhs)")
		}
		return &when{
			lhs: Q(v, vals...),
		}

	case ExpressionBuilder:
		return &when{
			lhs: v.BuildExpression(),
		}

	case LogicalExpression, ClauseExpression:
		var whenExpr = when{
			lhs: v.(Expression),
		}

		// vals are the then clause
		if len(vals) > 0 {
			if len(vals) > 1 {
				panic("only one value allowed when using a LogicalExpression or ClauseExpression as lhs")
			}

			switch expr := vals[0].(type) {
			case Expression:
				whenExpr.then = expr
			default:
				whenExpr.then = Value(expr)
			}
		}

		return &whenExpr

	case Expression:
		return &when{
			lhs: v,
		}

	default:
		panic("unsupported type for when clause, must be string or Expression")
	}
}

func (w *when) Then(value any) *when {
	if w.then != nil {
		panic("then value is already set, cannot set it again")
	}

	var exprs = expressionFromInterface[Expression](value, true)
	if len(exprs) == 0 || len(exprs) > 1 {
		panic("then value must be a single Expression")
	}

	w.then = exprs[0]
	return w
}

type CaseExpression struct {
	when []*when
	dflt Expression

	used bool
	inf  *ExpressionInfo
}

func Case(cases ...any) *CaseExpression {
	var dflt Expression
	var whenClauses = make([]*when, 0, len(cases))
	for _, c := range cases {
		switch v := c.(type) {
		case *when:
			whenClauses = append(whenClauses, v)
		case Expression:
			if dflt != nil {
				panic("default value already set, cannot add more when clauses")
			}
			if v == nil {
				panic("default value cannot be nil")
			}
			dflt = v
		case ExpressionBuilder:
			if dflt != nil {
				panic("default value already set, cannot add more when clauses")
			}
			var expr = v.BuildExpression()
			if expr == nil {
				panic("default value cannot be nil")
			}
			dflt = expr
		default:
			var exprs = expressionFromInterface[Expression](v, true)
			if len(exprs) == 0 || len(exprs) > 1 {
				panic("default value must be a single Expression")
			}
			if dflt != nil {
				panic("default value already set, cannot add more when clauses")
			}
			dflt = exprs[0]
		}
	}

	return &CaseExpression{
		when: whenClauses,
		dflt: dflt,
	}
}

func (c *CaseExpression) When(keyOrExpr interface{}, vals ...any) *CaseExpression {
	if c.used {
		panic("CaseExpression was already used, cannot add more when expressions")
	}

	var w = When(keyOrExpr, vals...)
	c.when = append(c.when, w)

	return c
}

func (c *CaseExpression) Then(value any) *CaseExpression {
	if len(c.when) == 0 {
		panic("no when clause defined, cannot set then value")
	}
	if c.used {
		panic("CaseExpression was already used, cannot set then value")
	}

	c.when[len(c.when)-1].Then(value)

	return c
}

func (c *CaseExpression) Default(value any) *CaseExpression {
	if c.used {
		panic("CaseExpression was already used, cannot set default value")
	}
	var exprs = expressionFromInterface[Expression](value, true)
	if len(exprs) == 0 || len(exprs) > 1 {
		panic("default value must be a single Expression")
	}
	c.dflt = exprs[0]
	return c
}

func (c *CaseExpression) Clone() Expression {
	var newWhen = make([]*when, len(c.when))
	for i, w := range c.when {
		if w.lhs == nil || w.then == nil {
			panic(fmt.Sprintf(
				"when clause at index %d is not fully defined (lhs=%v, then=%v)",
				i, w.lhs == nil, w.then == nil,
			))
		}

		var rhs Expression
		if w.rhs != nil {
			rhs = w.rhs.Clone()
		}

		newWhen[i] = &when{
			lhs:  w.lhs.Clone(),
			rhs:  rhs,
			then: w.then.Clone(),
		}
	}

	var dflt Expression
	if c.dflt != nil {
		dflt = c.dflt.Clone()
	}

	return &CaseExpression{
		when: newWhen,
		dflt: dflt,
		used: c.used,
	}
}

func (c *CaseExpression) Resolve(inf *ExpressionInfo) Expression {
	if inf.Model == nil || c.used {
		return c
	}

	var nC = c.Clone().(*CaseExpression)
	nC.used = true
	nC.inf = inf

	for i, w := range nC.when {
		var rhs Expression
		if w.rhs != nil {
			rhs = w.rhs.Resolve(inf)
		}

		nC.when[i] = &when{
			lhs:  w.lhs.Resolve(inf),
			rhs:  rhs,
			then: w.then.Resolve(inf),
		}
	}

	if nC.dflt != nil {
		nC.dflt = nC.dflt.Resolve(inf)
	}

	return nC
}

func (c *CaseExpression) SQL(sb *strings.Builder) []any {
	if !c.used {
		panic("CaseExpression was not resolved, cannot generate SQL")
	}

	var args = make([]any, 0)

	sb.WriteString("CASE ")

	for i, w := range c.when {
		if w.lhs == nil || w.then == nil {
			panic("when clause is not fully defined, lhs and then must be set")
		}

		if i > 0 {
			sb.WriteString(" ")
		}

		sb.WriteString("WHEN ")

		args = append(args, w.lhs.SQL(sb)...)

		if w.rhs != nil {
			sb.WriteString(" ")

			var inner strings.Builder
			args = append(args, w.rhs.SQL(&inner)...)

			var rhs, _ = c.inf.Lookups.FormatLogicalOpRHS(
				EQ, inner.String(),
			)
			sb.WriteString(rhs)
		}

		sb.WriteString(" THEN ")
		args = append(args, w.then.SQL(sb)...)
	}

	if c.dflt != nil {
		sb.WriteString(" ELSE ")
		args = append(args, c.dflt.SQL(sb)...)
	}

	sb.WriteString(" END")

	return args
}
