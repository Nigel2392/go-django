package expr_test

import (
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/queries/src/expr"
)

// SQL Generation 1
func TestRawExprSQL(t *testing.T) {
	info := getTestInfo()
	r := expr.Raw("![Age] = ?", 18)
	resolved := r.Resolve(info)
	var sb strings.Builder
	resolved.SQL(&sb)
	if !strings.Contains(sb.String(), fixSQL(info, "`test_model`.`age` = ?")) {
		t.Errorf("Unexpected Raw SQL generation: %s", sb.String())
	}
}

// SQL Generation 2
func TestFExprSQL(t *testing.T) {
	info := getTestInfo()
	f := expr.F("![Age] + ?[1] + ![Score]", 3)
	resolved := f.Resolve(info)
	var sb strings.Builder
	args := resolved.SQL(&sb)
	sql := sb.String()
	if !strings.Contains(sql, fixSQL(info, "`test_model`.`age` + ? + `test_model`.`score`")) {
		t.Errorf("Unexpected F SQL generation: %s", sql)
	}
	if len(args) != 1 || args[0] != 3 {
		t.Errorf("Expected arg 3, got %v", args)
	}
}

// Happy Path 1
func TestFExprMultipleArgs(t *testing.T) {
	info := getTestInfo()
	f := expr.F("![Age] > ? AND ![Score] < ?", 18, 100)
	resolved := f.Resolve(info)
	if resolved == nil {
		t.Fatalf("Failed to resolve FExpr")
	}
}

// Happy Path 2
func TestRawExprNot(t *testing.T) {
	info := getTestInfo()
	q := expr.Q("Age", 18)
	notR := q.Not(true)
	resolved := notR.Resolve(info)
	var sb strings.Builder
	resolved.SQL(&sb)
	if !strings.Contains(sb.String(), fixSQL(info, "NOT (`test_model`.`age` = ?)")) {
		t.Errorf("Unexpected NOT Raw SQL: %s", sb.String())
	}
}

// Unhappy Path 1
func TestFExprResolveInvalidField(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic when resolving FExpr with invalid field")
		}
	}()
	info := getTestInfo()
	f := expr.F("![InvalidField] = ?", 1)
	f.Resolve(info)
}

// Unhappy Path 2
func TestRawExprInvalidArgsCount(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic when indexing non-existent argument")
		}
	}()
	info := getTestInfo()
	// Using ?[2] but only providing 1 argument should panic during parse/resolve
	r := expr.Raw("![Age] = ?[2]", 1)
	r.Resolve(info)
}
