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
