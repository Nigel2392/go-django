package expr_test

import (
	"strings"
	"testing"
	"github.com/Nigel2392/go-django/queries/src/expr"
)

func TestExprOperatorSQLGeneration1(t *testing.T) {
	resolved := expr.EQ.Resolve(nil)
	var sb strings.Builder
	args := resolved.SQL(&sb)
	if len(args) != 0 {
		t.Errorf("Expected 0 args, got %d", len(args))
	}
	sql := sb.String()
	if !strings.Contains(sql, "=") {
		t.Errorf("Expected '=', got: %s", sql)
	}
}

func TestExprOperatorSQLGeneration2(t *testing.T) {
	resolved := expr.IN.Resolve(nil)
	var sb strings.Builder
	resolved.SQL(&sb)
	sql := sb.String()
	if !strings.Contains(sql, "IN") {
		t.Errorf("Expected 'IN', got: %s", sql)
	}
}

func TestExprOperatorHappyPath1(t *testing.T) {
	if string(expr.EQ) != "=" {
		t.Errorf("Expected '=', got %s", expr.EQ)
	}
}

func TestExprOperatorHappyPath2(t *testing.T) {
	cloned := expr.EQ.Clone()
	if cloned != expr.EQ {
		t.Errorf("Expected cloned operator to match")
	}
}

func TestExprOperatorUnhappyPath1(t *testing.T) {
	// Casting an invalid string to Operator will format as string, but won't panic.
	bad := expr.LogicalOp("INVALID_OP")
	if string(bad) != "INVALID_OP" {
		t.Errorf("Expected INVALID_OP")
	}
}

func TestExprOperatorUnhappyPath2(t *testing.T) {
	// A logical op cast test
	bad := expr.LogicalOp("NOT_A_LOGICAL_OP")
	if string(bad) != "NOT_A_LOGICAL_OP" {
		t.Errorf("Expected NOT_A_LOGICAL_OP")
	}
}
