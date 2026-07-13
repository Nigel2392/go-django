package expr_test

import (
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/queries/src/expr"
)

// SQL Generation 1
func TestExprChainAndSQL(t *testing.T) {
	info := getTestInfo()
	c := expr.Chain(expr.Field("Age"), expr.EQ, expr.Field("Name"))
	resolved := c.Resolve(info)
	var sb strings.Builder
	resolved.SQL(&sb)
	sql := sb.String()
	if !strings.Contains(sql, fixSQL(info, "`test_model`.`age` = `test_model`.`name`")) {
		t.Errorf("Unexpected Chain And SQL output: %s", sql)
	}
}

// SQL Generation 2
func TestExprChainOrSQL(t *testing.T) {
	info := getTestInfo()
	c := expr.Chain(expr.Field("Age"), expr.NE, expr.Field("Name"))
	resolved := c.Resolve(info)
	var sb strings.Builder
	resolved.SQL(&sb)
	sql := sb.String()
	if !strings.Contains(sql, fixSQL(info, "`test_model`.`age` != `test_model`.`name`")) {
		t.Errorf("Unexpected Chain Or SQL output: %s", sql)
	}
}

// Happy Path 1
func TestExprChainChaining(t *testing.T) {
	info := getTestInfo()
	c := expr.Chain(expr.Field("Age"), expr.EQ, expr.Field("Name"), expr.NE, expr.Field("Score"))
	resolved := c.Resolve(info)
	if resolved == nil {
		t.Fatalf("Failed to resolve complex chain")
	}
}

// Happy Path 2
func TestExprChainCondition(t *testing.T) {
	info := getTestInfo()
	c := expr.Chain(expr.Field("Age"), expr.EQ, expr.Value(18))
	resolved := c.Resolve(info)
	var sb strings.Builder
	args := resolved.SQL(&sb)
	sql := sb.String()
	if !strings.Contains(sql, fixSQL(info, "`test_model`.`age` = ?")) {
		t.Errorf("Unexpected Chain Condition output: %s", sql)
	}
	if len(args) != 1 || args[0] != 18 {
		t.Errorf("Expected arg 18, got %v", args)
	}
}

// Unhappy Path 1
func TestExprChainResolveInvalidAnd(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic when resolving chain with invalid field")
		}
	}()
	info := getTestInfo()
	c := expr.Chain(expr.Field("Age"), expr.EQ, expr.Field("InvalidField"))
	c.Resolve(info)
}

// Unhappy Path 2
func TestExprChainResolveInvalidOr(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic when resolving chain with invalid field in Or")
		}
	}()
	info := getTestInfo()
	c := expr.Chain(expr.Field("InvalidField"), expr.EQ, expr.Field("Name"))
	c.Resolve(info)
}
