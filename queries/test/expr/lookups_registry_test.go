package expr_test

import (
	"strings"
	"testing"
	
	"github.com/Nigel2392/go-django/queries/src/expr"
)

// SQL Generation 1
func TestLookupRegistrySQLGeneration1(t *testing.T) {
	// Custom lookup mapping using InLookup as a proxy since it's exported and works
	expr.RegisterLookup(&expr.InLookup{
		BaseLookup: expr.BaseLookup{
			Identifier: "custom_op",
		},
	})
	info := getTestInfo()
	q := expr.Q("Age__custom_op", []int{42})
	resolved := q.Resolve(info)
	var sb strings.Builder
	args := resolved.SQL(&sb)
	if !strings.Contains(sb.String(), fixSQL(info, "`test_model`.`age`")) || !strings.Contains(sb.String(), "IN") {
		t.Errorf("Unexpected Custom Lookup SQL: %s", sb.String())
	}
	if len(args) != 1 || args[0] != 42 {
		t.Errorf("Unexpected args: %v", args)
	}
}

// SQL Generation 2
func TestLookupRegistrySQLGeneration2(t *testing.T) {
	expr.RegisterLookup(&expr.InLookup{
		BaseLookup: expr.BaseLookup{
			Identifier: "is_cool",
		},
	})
	info := getTestInfo()
	q := expr.Q("Name__is_cool", []string{"yes", "no"})
	resolved := q.Resolve(info)
	var sb strings.Builder
	resolved.SQL(&sb)
	if !strings.Contains(sb.String(), fixSQL(info, "`test_model`.`name`")) || !strings.Contains(sb.String(), "IN") {
		t.Errorf("Unexpected Custom Lookup SQL: %s", sb.String())
	}
}

// Happy Path 1
func TestLookupRegistryHappyPath1(t *testing.T) {
	expr.RegisterLookup(&expr.InLookup{
		BaseLookup: expr.BaseLookup{
			Identifier: "custom_happy_1",
		},
	})
	info := getTestInfo()
	lookupFunc, err := expr.GetLookup(info, "custom_happy_1", "`test_model`.`age`", []any{[]int{1}})
	if err != nil {
		t.Fatalf("Failed to get lookup: %v", err)
	}
	var sb strings.Builder
	lookupFunc(&sb)
	if sb.String() == "" {
		t.Errorf("Expected SQL from lookup")
	}
}

// Happy Path 2
func TestLookupRegistryHappyPath2(t *testing.T) {
	expr.RegisterLookup(&expr.InLookup{
		BaseLookup: expr.BaseLookup{
			Identifier: "custom_happy_2",
		},
	})
	info := getTestInfo()
	lookupFunc, err := expr.GetLookup(info, "custom_happy_2", "`test_model`.`age`", []any{[]int{1}})
	if err != nil {
		t.Fatalf("Failed to get lookup: %v", err)
	}
	if lookupFunc == nil {
		t.Errorf("Expected non-nil lookup func")
	}
}

// Unhappy Path 1
func TestLookupRegistryUnhappyPath1(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic on nil lookup registration")
		}
	}()
	expr.RegisterLookup(nil)
}

// Unhappy Path 2
func TestLookupRegistryUnhappyPath2(t *testing.T) {
	info := getTestInfo()
	_, err := expr.GetLookup(info, "non_existent_lookup", "`test_model`.`age`", []any{1})
	if err == nil {
		t.Errorf("Expected error when getting non-existent lookup")
	}
}
