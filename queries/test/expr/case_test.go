package expr_test

import (
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/djester/testdb"
	"github.com/Nigel2392/go-django/queries/src/expr"
)

func TestCaseWhenSQL(t *testing.T) {
	info := getTestInfo()

	w := expr.When(expr.Raw("![Age] = ?", 18)).Then("adult")
	c := expr.Case(w, expr.Value("minor"))
	resolved := c.Resolve(info)
	var sb strings.Builder
	args := resolved.SQL(&sb)
	sql := sb.String()

	t.Logf("Generated SQL: %q", sql)
	
	expectedSQL := "CASE WHEN `test_model`.`age` = ? THEN ? ELSE ? END"
	if testdb.ENGINE == "postgres" {
		expectedSQL = "CASE WHEN `test_model`.`age` = ? THEN ?::TEXT ELSE ?::TEXT END"
	}
	if !strings.Contains(sql, fixSQL(info, expectedSQL)) {
		t.Errorf("Unexpected Case SQL generation: %q", sql)
	}
	if len(args) != 3 {
		t.Errorf("Expected 3 arguments for Case expression, got %d", len(args))
	}
}

func TestCaseWhenComplex(t *testing.T) {
	info := getTestInfo()
	f1 := expr.Q("Age__gt", 18)
	w := expr.When(f1).Then("adult")
	c := expr.Case(w, expr.Value("minor"))
	resolved := c.Resolve(info)
	var sb strings.Builder
	args := resolved.SQL(&sb)
	sql := sb.String()

	expectedSQL := "CASE WHEN `test_model`.`age` > ? THEN ? ELSE ? END"
	if testdb.ENGINE == "postgres" {
		expectedSQL = "CASE WHEN `test_model`.`age` > ? THEN ?::TEXT ELSE ?::TEXT END"
	}
	if !strings.Contains(sql, fixSQL(info, expectedSQL)) {
		t.Errorf("Unexpected Case SQL generation from Q: %s", sql)
	}
	if args[0] != 18 || args[1] != "adult" || args[2] != "minor" {
		t.Errorf("Unexpected args generated: %v", args)
	}
}

func TestCaseNoDefault(t *testing.T) {
	info := getTestInfo()
	w := expr.When(expr.Q("Age__gt", 18)).Then("adult")
	c := expr.Case(w)
	resolved := c.Resolve(info)
	var sb strings.Builder
	resolved.SQL(&sb)
	sql := sb.String()

	if strings.Contains(sql, "ELSE") {
		t.Errorf("Unexpected ELSE in Case without default: %s", sql)
	}
}

func TestWhenInvalidArgs(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Expected panic on empty When vals with string key")
		}
	}()
	_ = expr.When("Age") // Panics because string key implies Q object which needs 2 args
}
