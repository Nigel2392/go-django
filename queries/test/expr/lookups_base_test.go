package expr_test

import (
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/queries/src/expr"
)

// SQL Generation 1
func TestLookupExactSQL(t *testing.T) {
	info := getTestInfo()
	q := expr.Q("Age__exact", 18)
	resolved := q.Resolve(info)
	var sb strings.Builder
	args := resolved.SQL(&sb)
	if !strings.Contains(sb.String(), fixSQL(info, "`test_model`.`age` = ?")) {
		t.Errorf("Unexpected Exact Lookup SQL: %s", sb.String())
	}
	if len(args) != 1 || args[0] != 18 {
		t.Errorf("Expected arg 18, got %v", args)
	}
}

// SQL Generation 2
func TestLookupInSQL(t *testing.T) {
	info := getTestInfo()
	q := expr.Q("Age__in", []int{18, 19, 20})
	resolved := q.Resolve(info)
	var sb strings.Builder
	args := resolved.SQL(&sb)
	if !strings.Contains(sb.String(), fixSQL(info, "`test_model`.`age` IN (?, ?, ?)")) {
		t.Errorf("Unexpected In Lookup SQL: %s", sb.String())
	}
	if len(args) != 3 {
		t.Errorf("Expected 3 args, got %d", len(args))
	}
}

// Happy Path 1
func TestLookupContainsSQL(t *testing.T) {
	info := getTestInfo()
	q := expr.Q("Name__contains", "John")
	resolved := q.Resolve(info)
	var sb strings.Builder
	args := resolved.SQL(&sb)
	
	// Just checking for basic LIKE/ILIKE substring to avoid dialect hell
	if !strings.Contains(strings.ToUpper(sb.String()), "LIKE") {
		t.Errorf("Unexpected Contains Lookup SQL: %s", sb.String())
	}
	if len(args) != 1 || args[0] != "%John%" {
		t.Errorf("Expected pattern %%John%%, got %v", args)
	}
}

// Happy Path 2
func TestLookupIsNullSQL(t *testing.T) {
	info := getTestInfo()
	q := expr.Q("Name__isnull", true)
	resolved := q.Resolve(info)
	var sb strings.Builder
	resolved.SQL(&sb)
	if !strings.Contains(sb.String(), fixSQL(info, "`test_model`.`name` IS NULL")) {
		t.Errorf("Unexpected IsNull Lookup SQL: %s", sb.String())
	}
}

// Unhappy Path 1
func TestLookupInvalidOperator(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic when using unregistered lookup")
		}
	}()
	info := getTestInfo()
	q := expr.Q("Age__invalid_lookup", 18)
	q.Resolve(info)
}

// Unhappy Path 2
func TestLookupInInvalidArg(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic when using __in with no arguments")
		}
	}()
	info := getTestInfo()
	q := expr.Q("Age__in") // expects at least one value
	q.Resolve(info)
}

func TestBaseLookupResolvePanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic when ResolveFunc is nil")
		}
	}()
	l := &expr.BaseLookup{Identifier: "test_lookup"}
	l.Resolve(nil, nil, nil)
}

func TestRangeLookupCoverage(t *testing.T) {
	info := getTestInfo()
	l := &expr.RangeLookup{
		BaseLookup: expr.BaseLookup{
			Identifier: "range",
		},
	}
	min, max := l.Arity()
	if min != 2 || max != 2 {
		t.Errorf("Expected arity 2,2, got %d,%d", min, max)
	}

	f1 := expr.Field("Age")
	r1 := f1.Resolve(info)
	
	fn := l.Resolve(info, r1, []any{10, 20})
	var sb strings.Builder
	args := fn(&sb)
	if len(args) != 2 || args[0] != 10 || args[1] != 20 {
		t.Errorf("Expected 10 and 20 in args")
	}
	if !strings.Contains(sb.String(), "BETWEEN ? AND ?") {
		t.Errorf("Expected BETWEEN ? AND ?, got %s", sb.String())
	}

	// Test panic
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic with bad arg count")
		}
	}()
	fnBad := l.Resolve(info, r1, []any{10})
	var sb2 strings.Builder
	fnBad(&sb2)
}
