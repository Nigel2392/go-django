package expr_test

import (
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/queries/src/expr"
)

func TestLogicalQ(t *testing.T) {
	info := getTestInfo()
	q := expr.Q("Name", "John")
	resolved := q.Resolve(info)

	var sb strings.Builder
	args := resolved.SQL(&sb)
	if sb.String() != fixSQL(info, "`test_model`.`name` = ?") {
		t.Errorf("Unexpected SQL output for basic Q: %s", sb.String())
	}
	if len(args) != 1 || args[0] != "John" {
		t.Errorf("Unexpected args: %v", args)
	}
}

func TestLogicalAndChaining(t *testing.T) {
	info := getTestInfo()
	q := expr.And(expr.Q("Age__gt", 18), expr.Q("Name", "John"))
	resolved := q.Resolve(info)

	var sb strings.Builder
	args := resolved.SQL(&sb)
	sql := sb.String()

	if !strings.Contains(sql, fixSQL(info, "`test_model`.`age` > ?")) || !strings.Contains(sql, "AND") || !strings.Contains(sql, fixSQL(info, "`test_model`.`name` = ?")) {
		t.Errorf("Unexpected SQL output for And: %s", sql)
	}
	if len(args) != 2 {
		t.Errorf("Unexpected args count: %d", len(args))
	}
}

func TestLogicalNot(t *testing.T) {
	info := getTestInfo()
	q := expr.Q("Name", "John").Not(true)
	resolved := q.Resolve(info)

	var sb strings.Builder
	resolved.SQL(&sb)
	sql := sb.String()

	if !strings.HasPrefix(sql, "NOT (") || !strings.HasSuffix(sql, ")") {
		t.Errorf("Expected NOT wrapper, got: %s", sql)
	}
}

func TestLogicalQInvalidArgs(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Expected panic on Q with invalid field type")
		}
	}()
	_ = expr.Expr(123, "exact", 42) // Panics because 123 is not a valid field
}

func TestLogicalOperatorsCoverage(t *testing.T) {
	// Test node methods
	n := expr.Q("Name", "John")
	if n.IsNot() {
		t.Errorf("Expected Not to be false")
	}
	n.Not(true)
	if !n.IsNot() {
		t.Errorf("Expected Not to be true")
	}

	// Test ExprGroup properties
	g := expr.And(expr.Q("Age__gt", 18), expr.Q("Name", "John"))
	if g.IsNot() {
		t.Errorf("Expected Not to be false")
	}
	g.Not(true)
	if !g.IsNot() {
		t.Errorf("Expected Not to be true")
	}
	if g.Operator() != expr.OpAnd {
		t.Errorf("Expected OpAnd, got %v", g.Operator())
	}
	if len(g.Unwrap()) != 2 {
		t.Errorf("Expected 2 children, got %d", len(g.Unwrap()))
	}

	info := getTestInfo()
	gAnd := g.And(expr.Q("Score", 100))
	resolvedAnd := gAnd.Resolve(info)
	var sbAnd strings.Builder
	resolvedAnd.SQL(&sbAnd)
	if !strings.Contains(sbAnd.String(), "AND") {
		t.Errorf("Expected AND, got %s", sbAnd.String())
	}

	gOr := g.Or(expr.Q("Score", 100))
	resolvedOr := gOr.Resolve(info)
	var sbOr strings.Builder
	resolvedOr.SQL(&sbOr)
	if !strings.Contains(sbOr.String(), "OR") {
		t.Errorf("Expected OR, got %s", sbOr.String())
	}

	if len(gAnd.(*expr.ExprGroup).Unwrap()) != 2 {
		t.Errorf("Expected 2 children after And(), got %d", len(gAnd.(*expr.ExprGroup).Unwrap()))
	}

	if len(gOr.(*expr.ExprGroup).Unwrap()) != 2 {
		t.Errorf("Expected 2 children after Or(), got %d", len(gOr.(*expr.ExprGroup).Unwrap()))
	}

	emptyGroup := &expr.ExprGroup{}
	emptyAnd := emptyGroup.And(expr.Q("A", 1))
	if emptyAnd.(*expr.ExprGroup).Operator() != expr.OpAnd {
		t.Errorf("Expected OpAnd on empty group")
	}
	emptyOr := emptyGroup.Or(expr.Q("A", 1))
	if emptyOr.(*expr.ExprGroup).Operator() != expr.OpOr {
		t.Errorf("Expected OpOr on empty group")
	}
}

func TestLogicalScope(t *testing.T) {
	info := getTestInfo()
	chain := expr.Logical("Age")
	q := chain.Scope(expr.EQ, expr.Q("Name", "John"))
	resolved := q.Resolve(info)
	var sb strings.Builder
	resolved.SQL(&sb)
	if !strings.Contains(sb.String(), fixSQL(info, "`test_model`.`age` = ")) || !strings.Contains(sb.String(), fixSQL(info, "`test_model`.`name`")) {
		t.Errorf("Expected Scope to contain both fields, got: %s", sb.String())
	}
}

func TestLogicalOpsMethods(t *testing.T) {
	ops := []struct {
		fn       func(l expr.LogicalExpression, v interface{}) expr.LogicalExpression
		expected string
	}{
		{func(l expr.LogicalExpression, v interface{}) expr.LogicalExpression { return l.NE(v) }, "!="},
		{func(l expr.LogicalExpression, v interface{}) expr.LogicalExpression { return l.GT(v) }, ">"},
		{func(l expr.LogicalExpression, v interface{}) expr.LogicalExpression { return l.LT(v) }, "<"},
		{func(l expr.LogicalExpression, v interface{}) expr.LogicalExpression { return l.GTE(v) }, ">="},
		{func(l expr.LogicalExpression, v interface{}) expr.LogicalExpression { return l.LTE(v) }, "<="},
		{func(l expr.LogicalExpression, v interface{}) expr.LogicalExpression { return l.MUL(v) }, "*"},
		{func(l expr.LogicalExpression, v interface{}) expr.LogicalExpression { return l.DIV(v) }, "/"},
		{func(l expr.LogicalExpression, v interface{}) expr.LogicalExpression { return l.MOD(v) }, "%"},
		{func(l expr.LogicalExpression, v interface{}) expr.LogicalExpression { return l.BITAND(v) }, "&"},
		{func(l expr.LogicalExpression, v interface{}) expr.LogicalExpression { return l.BITOR(v) }, "|"},
		{func(l expr.LogicalExpression, v interface{}) expr.LogicalExpression { return l.BITXOR(v) }, "^"},
		{func(l expr.LogicalExpression, v interface{}) expr.LogicalExpression { return l.BITLSH(v) }, "<<"},
		{func(l expr.LogicalExpression, v interface{}) expr.LogicalExpression { return l.BITRSH(v) }, ">>"},
	}

	info := getTestInfo()
	v1 := 18

	for _, tc := range ops {
		chain := expr.Logical("Age")
		opExpr := tc.fn(chain, v1)
		resolved := opExpr.Resolve(info)
		var sb strings.Builder
		resolved.SQL(&sb)
		if !strings.Contains(sb.String(), tc.expected) {
			t.Errorf("Expected %s, got %s", tc.expected, sb.String())
		}
	}

	// BITNOT is unary
	chain := expr.Logical("Age")
	bnot := chain.BITNOT(nil)
	resolved := bnot.Resolve(info)
	var sb strings.Builder
	resolved.SQL(&sb)
	if !strings.Contains(sb.String(), "~") {
		t.Errorf("Expected ~, got %s", sb.String())
	}
}
