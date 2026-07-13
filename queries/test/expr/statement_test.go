package expr_test

import (
	"strings"
	"testing"
	"github.com/Nigel2392/go-django/queries/src/expr"
)

// SQL Generation 1
func TestStatementParseSQL1(t *testing.T) {
	info := getTestInfo()
	stmt := expr.ParseExprStatement("![Age] = ?[1]", []any{18})
	resolved := stmt.Resolve(info)
	sqlStr, args := resolved.SQL()
	if !strings.Contains(sqlStr, fixSQL(info, "`test_model`.`age` = ?")) {
		t.Errorf("Unexpected ExpressionStatement SQL: %s", sqlStr)
	}
	if len(args) != 1 || args[0] != 18 {
		t.Errorf("Unexpected args: %v", args)
	}
}

// SQL Generation 2
func TestStatementParseSQL2(t *testing.T) {
	info := getTestInfo()
	stmt := expr.ParseExprStatement("?[1] + ?[2] = ![Score]", []any{5, 10})
	resolved := stmt.Resolve(info)
	sqlStr, args := resolved.SQL()
	if !strings.Contains(sqlStr, fixSQL(info, "? + ? = `test_model`.`score`")) {
		t.Errorf("Unexpected ExpressionStatement SQL: %s", sqlStr)
	}
	if len(args) != 2 || args[0] != 5 || args[1] != 10 {
		t.Errorf("Unexpected args: %v", args)
	}
}

// Happy Path 1
func TestStatementParseSelfTable(t *testing.T) {
	info := getTestInfo()
	stmt := expr.ParseExprStatement("SELECT * FROM table(SELF) WHERE ![Age] = ?", []any{18})
	resolved := stmt.Resolve(info)
	sqlStr, _ := resolved.SQL()
	if !strings.Contains(sqlStr, fixSQL(info, "`test_model`")) {
		t.Errorf("Unexpected SELF table resolution SQL: %s", sqlStr)
	}
}

// Happy Path 2
func TestStatementFieldCount(t *testing.T) {
	stmt := expr.ParseExprStatement("![Age] = ? AND ![Name] = ?", []any{18, "John"})
	fields := stmt.Raw("field")
	if len(fields) != 2 {
		t.Errorf("Expected 2 fields parsed, got %d", len(fields))
	}
	if len(fields) == 2 && (fields[0] != "Age" || fields[1] != "Name") {
		t.Errorf("Unexpected parsed fields: %v", fields)
	}
}

// Unhappy Path 1
func TestStatementResolveInvalidField(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic resolving unmapped field")
		}
	}()
	info := getTestInfo()
	stmt := expr.ParseExprStatement("![UnknownField] = ?", []any{18})
	stmt.Resolve(info)
}

// Unhappy Path 2
func TestStatementMissingArgs(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic parsing ?[2] with only 1 arg")
		}
	}()
	info := getTestInfo()
	stmt := expr.ParseExprStatement("![Age] = ?[2]", []any{18}) // Wants arg at index 1 (1-based index 2)
	stmt.Resolve(info)
}
