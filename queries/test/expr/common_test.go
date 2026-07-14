package expr_test

import (
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/djester/testdb"
	"github.com/Nigel2392/go-django/queries/src/expr"
)

// SQL Generation 1
func TestStringExprSQLGeneration(t *testing.T) {
	s := expr.String("SELECT * FROM table")
	var sb strings.Builder
	args := s.SQL(&sb)
	if sb.String() != "SELECT * FROM table" {
		t.Errorf("Unexpected StringExpr output: %s", sb.String())
	}
	if len(args) != 0 {
		t.Errorf("Expected 0 args, got %v", args)
	}
}

// SQL Generation 2
func TestFieldExprSQLGeneration(t *testing.T) {
	info := getTestInfo()
	f := expr.Field("Name")
	resolved := f.Resolve(info)
	var sb strings.Builder
	resolved.SQL(&sb)
	if !strings.Contains(sb.String(), fixSQL(info, "`test_model`.`name`")) {
		t.Errorf("Unexpected FieldExpr output: %s", sb.String())
	}
}

// Happy Path 1
func TestValueExprResolve(t *testing.T) {
	info := getTestInfo()
	v := expr.Value("hello")
	resolved := v.Resolve(info)
	var sb strings.Builder
	args := resolved.SQL(&sb)
	expectedSQL := fixSQL(info, "?")
	if testdb.ENGINE == "postgres" {
		expectedSQL = fixSQL(info, "?::TEXT")
	}
	if sb.String() != expectedSQL {
		t.Errorf("Expected ?, got %s", sb.String())
	}
	if len(args) != 1 || args[0] != "hello" {
		t.Errorf("Expected arg 'hello', got %v", args)
	}
}

// Happy Path 2
func TestNamedValueResolve(t *testing.T) {
	info := getTestInfo()
	v := expr.As("Age", expr.V(30))
	resolved := v.Resolve(info)
	if resolved == nil {
		t.Fatalf("Failed to resolve NamedValue")
	}
	// NamedValue acts mostly like Value but holds field name reference.
}

// Unhappy Path 1
func TestFieldExprResolveInvalidField(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic when resolving invalid field")
		}
	}()
	info := getTestInfo()
	f := expr.Field("InvalidField")
	f.Resolve(info)
}

// Unhappy Path 2
func TestNamedValueResolveInvalidField(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic when resolving NamedValue with invalid field")
		}
	}()
	info := getTestInfo()
	info.ForUpdate = true
	v := expr.As("InvalidField", expr.V(30))
	v.Resolve(info)
}

func TestStringCoverage(t *testing.T) {
	s := expr.String("test")
	if s.String() != "test" {
		t.Errorf("Expected test, got %s", s.String())
	}
	s2 := s.Clone()
	if s2.(expr.String) != "test" {
		t.Errorf("Expected cloned test, got %s", s2)
	}
}

func TestValuesCoverage(t *testing.T) {
	info := getTestInfo()
	v := expr.Values(1, 2, 3)
	v2 := v.Clone()
	
	resolved := v2.Resolve(info)
	var sb strings.Builder
	args := resolved.SQL(&sb)
	
	if len(args) != 3 {
		t.Errorf("Expected 3 args, got %d", len(args))
	}
	
	// Also test unhappy path where values are empty
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic on empty values")
		}
	}()
	expr.Values()
}
