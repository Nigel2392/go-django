package expr

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Nigel2392/go-django/src/core/attrs"
)

func Q(fieldLookup string, value ...any) *ExprNode {
	var (
		split  = strings.SplitN(fieldLookup, "__", 2)
		lookup = DEFAULT_LOOKUP
		field  string
	)

	if len(split) > 1 {
		field = split[0]
		lookup = split[1]
	} else {
		field = fieldLookup
	}

	return Expr(field, lookup, value...)
}

func And(exprs ...Expression) *ExprGroup {
	return &ExprGroup{children: exprs, op: OpAnd, wrap: true}
}

func Or(exprs ...Expression) *ExprGroup {
	return &ExprGroup{children: exprs, op: OpOr, wrap: true}
}

type ExprNode struct {
	sql    func(sb *strings.Builder) []any
	args   []any
	not    bool
	model  attrs.Definer
	field  NamedExpression
	lookup string
	used   bool
}

func Expr(field any, operation string, value ...any) *ExprNode {
	var exprs = expressionFromInterface[NamedExpression](field, false)
	if len(exprs) == 0 {
		panic(fmt.Errorf("field must be a string or an expression, got %T", field))
	}
	if len(exprs) > 1 {
		panic(fmt.Errorf("field must be a single string or expression, got %d expressions", len(exprs)))
	}
	return &ExprNode{
		args:   value,
		field:  exprs[0],
		lookup: operation,
	}
}

func (e *ExprNode) Resolve(inf *ExpressionInfo) Expression {
	if inf.Model == nil {
		panic("model is nil")
	}

	if e.used {
		return e
	}

	var nE = e.Clone().(*ExprNode)
	nE.used = true
	nE.model = inf.Model

	var err error
	nE.field = nE.field.Resolve(inf).(NamedExpression)
	nE.sql, err = GetLookup(
		inf, nE.lookup, nE.field, nE.args,
	)
	if err != nil {
		panic(err)
	}

	return nE
}

func (e *ExprNode) SQL(sb *strings.Builder) []any {
	if e.not {
		sb.WriteString("NOT (")
	}

	var args = e.sql(sb)

	if e.not {
		sb.WriteString(")")
	}
	return args
}

func (e *ExprNode) Not(not bool) ClauseExpression {
	e.not = not
	return e
}

func (e *ExprNode) IsNot() bool {
	return e.not
}

func (e *ExprNode) And(exprs ...Expression) ClauseExpression {
	return &ExprGroup{children: append([]Expression{e}, exprs...), op: OpAnd, wrap: true}
}

func (e *ExprNode) Or(exprs ...Expression) ClauseExpression {
	return &ExprGroup{children: append([]Expression{e}, exprs...), op: OpOr, wrap: true}
}

func (e *ExprNode) Clone() Expression {
	return &ExprNode{
		sql:    e.sql,
		args:   e.args,
		not:    e.not,
		field:  e.field,
		lookup: e.lookup,
		model:  e.model,
		used:   e.used,
	}
}

// ExprGroup
type ExprGroup struct {
	children []Expression
	op       ExprOp
	not      bool
	wrap     bool
}

func (g *ExprGroup) SQL(sb *strings.Builder) []any {
	if g.not {
		sb.WriteString("NOT ")
	}
	if g.wrap {
		sb.WriteString("(")
	}
	var args = make([]any, 0)
	for i, child := range g.children {
		if i > 0 {
			if g.op != "" {
				sb.WriteString(" ")
				sb.WriteString(string(g.op))
				sb.WriteString(" ")
			}
		}

		args = append(args, child.SQL(sb)...)
	}
	if g.wrap {
		sb.WriteString(")")
	}
	return args
}

func (g *ExprGroup) Not(not bool) ClauseExpression {
	g.not = not
	return g
}

func (g *ExprGroup) IsNot() bool {
	return g.not
}

func (g *ExprGroup) And(exprs ...Expression) ClauseExpression {
	return &ExprGroup{children: append([]Expression{g}, exprs...), op: OpAnd, wrap: true}
}

func (g *ExprGroup) Or(exprs ...Expression) ClauseExpression {
	return &ExprGroup{children: append([]Expression{g}, exprs...), op: OpOr, wrap: true}
}

func (g *ExprGroup) Clone() Expression {
	clone := make([]Expression, len(g.children))
	for i, c := range g.children {
		clone[i] = c.Clone()
	}
	return &ExprGroup{
		children: clone,
		op:       g.op,
		not:      g.not,
		wrap:     g.wrap,
	}
}

func (g *ExprGroup) Resolve(inf *ExpressionInfo) Expression {
	var gClone = g.Clone().(*ExprGroup)
	for i, e := range gClone.children {
		gClone.children[i] = e.Resolve(inf)
	}
	return gClone
}

type logicalChainExpr struct {
	fieldName string
	field     *ResolvedField
	used      bool
	forUpdate bool
	inner     []Expression
}

func newChainExpr(expr NamedExpression) *logicalChainExpr {
	if expr == nil {
		panic(fmt.Errorf("cannot create logicalChainExpr with nil expression"))
	}

	return &logicalChainExpr{
		fieldName: expr.FieldName(),
		field:     nil,
		used:      false,
		forUpdate: false,
		inner:     []Expression{expr},
	}
}

func Logical(expr ...any) LogicalExpression {
	if len(expr) == 0 {
		panic(fmt.Errorf("logicalChainExpr requires at least one inner expression"))
	}

	var fieldName string
	var inner = make([]Expression, 0, len(expr))
	for i, e := range expr {

		if exprBuilder, ok := e.(ExpressionBuilder); ok {
			e = exprBuilder.BuildExpression()
		}

		if n, ok := e.(NamedExpression); ok && i == 0 {
			fieldName = n.FieldName()
		}

		inner = append(
			inner,
			expressionFromInterface[Expression](e, false)...,
		)
	}

	return &logicalChainExpr{
		fieldName: fieldName,
		used:      false,
		forUpdate: false,
		inner:     inner,
	}
}

func (l *logicalChainExpr) FieldName() string {
	if len(l.inner) == 0 {
		return ""
	}
	if l.fieldName != "" {
		return l.fieldName
	}
	if l.field != nil && l.field.Field != "" {
		return l.field.Field
	}
	for _, expr := range l.inner {
		if namer, ok := expr.(NamedExpression); ok {
			var name = namer.FieldName()
			if name != "" {
				return name
			}
		}
	}
	return ""
}

func (l *logicalChainExpr) SQL(sb *strings.Builder) []any {
	if len(l.inner) == 0 {
		panic(fmt.Errorf("SQL logicalChainExpr has no inner expressions"))
	}
	var args = make([]any, 0)
	//if l.field != nil {
	//	if l.field.SQLText != "" {
	//		sb.WriteString(l.field.SQLText)
	//	}
	//	if l.forUpdate && l.field.SQLText != "" {
	//		sb.WriteString(" = ")
	//	}
	//	args = append(args, l.field.SQLArgs...)
	//}
	for _, inner := range l.inner {
		args = append(args, inner.SQL(sb)...)
	}
	return args
}

func (l *logicalChainExpr) Clone() Expression {
	var inner = slices.Clone(l.inner)
	for i := range inner {
		inner[i] = inner[i].Clone()
	}
	return &logicalChainExpr{
		fieldName: l.fieldName,
		field:     l.field,
		used:      l.used,
		forUpdate: l.forUpdate,
		inner:     inner,
	}
}

func (l *logicalChainExpr) Resolve(inf *ExpressionInfo) Expression {
	if inf.Model == nil || l.used {
		return l
	}
	var nE = l.Clone().(*logicalChainExpr)
	nE.used = true
	nE.forUpdate = inf.ForUpdate

	if nE.fieldName != "" {
		nE.field = inf.ResolveExpressionField(nE.fieldName)
	}

	if len(nE.inner) > 0 {
		for i, inner := range nE.inner {
			nE.inner[i] = inner.Resolve(inf)
		}
	}

	return nE
}

func (l *logicalChainExpr) Scope(op LogicalOp, expr Expression) LogicalExpression {
	return &logicalChainExpr{
		fieldName: l.fieldName,
		used:      l.used,
		forUpdate: l.forUpdate,
		inner: append(slices.Clone(l.inner), op, &ExprGroup{
			children: []Expression{expr},
			op:       "",
		}),
	}
}

func (l *logicalChainExpr) chain(op LogicalOp, key interface{}, vals ...interface{}) LogicalExpression {
	var (
		copyExprs = slices.Clone(l.inner)
	)
	copyExprs = append(copyExprs, op)

	if key != nil {
		copyExprs = append(
			copyExprs,
			expressionFromInterface[Expression](key, false)...,
		)
	}

	for _, val := range vals {
		copyExprs = append(
			copyExprs,
			expressionFromInterface[Expression](val, false)...,
		)
	}

	return &logicalChainExpr{
		fieldName: l.fieldName,
		used:      l.used,
		forUpdate: l.forUpdate,
		inner:     copyExprs,
	}
}

func (l *logicalChainExpr) EQ(key interface{}, vals ...interface{}) LogicalExpression {
	return l.chain(EQ, key, vals...)
}
func (l *logicalChainExpr) NE(key interface{}, vals ...interface{}) LogicalExpression {
	return l.chain(NE, key, vals...)
}
func (l *logicalChainExpr) GT(key interface{}, vals ...interface{}) LogicalExpression {
	return l.chain(GT, key, vals...)
}
func (l *logicalChainExpr) LT(key interface{}, vals ...interface{}) LogicalExpression {
	return l.chain(LT, key, vals...)
}
func (l *logicalChainExpr) GTE(key interface{}, vals ...interface{}) LogicalExpression {
	return l.chain(GTE, key, vals...)
}
func (l *logicalChainExpr) LTE(key interface{}, vals ...interface{}) LogicalExpression {
	return l.chain(LTE, key, vals...)
}
func (l *logicalChainExpr) ADD(key interface{}, vals ...interface{}) LogicalExpression {
	return l.chain(ADD, key, vals...)
}
func (l *logicalChainExpr) SUB(key interface{}, vals ...interface{}) LogicalExpression {
	return l.chain(SUB, key, vals...)
}
func (l *logicalChainExpr) MUL(key interface{}, vals ...interface{}) LogicalExpression {
	return l.chain(MUL, key, vals...)
}
func (l *logicalChainExpr) DIV(key interface{}, vals ...interface{}) LogicalExpression {
	return l.chain(DIV, key, vals...)
}
func (l *logicalChainExpr) MOD(key interface{}, vals ...interface{}) LogicalExpression {
	return l.chain(MOD, key, vals...)
}
func (l *logicalChainExpr) BITAND(key interface{}, vals ...interface{}) LogicalExpression {
	return l.chain(BITAND, key, vals...)
}
func (l *logicalChainExpr) BITOR(key interface{}, vals ...interface{}) LogicalExpression {
	return l.chain(BITOR, key, vals...)
}
func (l *logicalChainExpr) BITXOR(key interface{}, vals ...interface{}) LogicalExpression {
	return l.chain(BITXOR, key, vals...)
}
func (l *logicalChainExpr) BITLSH(key interface{}, vals ...interface{}) LogicalExpression {
	return l.chain(BITLSH, key, vals...)
}
func (l *logicalChainExpr) BITRSH(key interface{}, vals ...interface{}) LogicalExpression {
	return l.chain(BITRSH, key, vals...)
}
func (l *logicalChainExpr) BITNOT(key interface{}, vals ...interface{}) LogicalExpression {
	return l.chain(BITNOT, key, vals...)
}
