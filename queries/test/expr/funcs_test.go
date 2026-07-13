package expr_test

import (
	"strings"
	"testing"
	"github.com/Nigel2392/go-django/queries/src/expr"
)

// SQL Generation 1
func TestFuncUpperSQL(t *testing.T) {
	info := getTestInfo()
	f := expr.UPPER(expr.Field("Name"))
	resolved := f.Resolve(info)
	var sb strings.Builder
	resolved.SQL(&sb)
	if sql := sb.String(); !strings.Contains(sql, fixSQL(info, "UPPER(`test_model`.`name`)")) {
		t.Errorf("Unexpected UPPER SQL: %s", sql)
	}
}

// SQL Generation 2
func TestFuncLowerSQL(t *testing.T) {
	info := getTestInfo()
	f := expr.LOWER(expr.Field("Name"))
	resolved := f.Resolve(info)
	var sb strings.Builder
	resolved.SQL(&sb)
	if sql := sb.String(); !strings.Contains(sql, fixSQL(info, "LOWER(`test_model`.`name`)")) {
		t.Errorf("Unexpected LOWER SQL: %s", sql)
	}
}

// Happy Path 1
func TestFuncLengthResolve(t *testing.T) {
	info := getTestInfo()
	f := expr.LENGTH(expr.Field("Name"))
	resolved := f.Resolve(info)
	var sb strings.Builder
	resolved.SQL(&sb)
	if sql := sb.String(); !strings.Contains(sql, fixSQL(info, "LENGTH(`test_model`.`name`)")) {
		t.Errorf("Unexpected LENGTH SQL: %s", sql)
	}
}

// Happy Path 2
func TestFuncNowResolve(t *testing.T) {
	info := getTestInfo()
	// NOW doesn't usually take args, testing basic func invocation
	f := expr.NOW()
	resolved := f.Resolve(info)
	if resolved == nil {
		t.Fatalf("Failed to resolve NOW")
	}
}

// Unhappy Path 1
func TestFuncInvalidArgsCount(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic when resolving function with incorrect arg count")
		}
	}()
	info := getTestInfo()
	// UPPER expects 1 arg, pass invalid arg
	f := expr.UPPER(123)
	f.Resolve(info)
}

// Unhappy Path 2
func TestFuncInvalidArgsCount2(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic when resolving SUBSTR with missing args")
		}
	}()
	info := getTestInfo()
	// SUBSTR expects 3 args. Passing invalid type to panic
	f := expr.SUBSTR(123, nil, nil)
	f.Resolve(info)
}
