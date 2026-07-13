package expr_test

import (
	"strings"
	"testing"
	"github.com/Nigel2392/go-django/queries/src/expr"
)

// SQL Generation 1
func TestUtilsExpressSQL1(t *testing.T) {
	info := getTestInfo()
	// Express creates a list of expressions from arguments.
	ce := expr.Express("Age__gt", 18)
	if len(ce) != 1 {
		t.Fatalf("Expected 1 expression, got %d", len(ce))
	}
	resolved := ce[0].Resolve(info)
	var sb strings.Builder
	args := resolved.SQL(&sb)
	if !strings.Contains(sb.String(), fixSQL(info, "`test_model`.`age` > ?")) {
		t.Errorf("Unexpected Express SQL: %s", sb.String())
	}
	if len(args) != 1 || args[0] != 18 {
		t.Errorf("Unexpected args: %v", args)
	}
}

// SQL Generation 2
func TestUtilsExpressSQL2(t *testing.T) {
	info := getTestInfo()
	// Express map
	ce := expr.Express(map[string]any{
		"Name": "John",
	})
	if len(ce) != 1 {
		t.Fatalf("Expected 1 expression, got %d", len(ce))
	}
	resolved := ce[0].Resolve(info)
	var sb strings.Builder
	args := resolved.SQL(&sb)
	if !strings.Contains(sb.String(), fixSQL(info, "`test_model`.`name` = ?")) {
		t.Errorf("Unexpected Express Map SQL: %s", sb.String())
	}
	if len(args) != 1 || args[0] != "John" {
		t.Errorf("Unexpected args: %v", args)
	}
}

// Happy Path 1
func TestUtilsExpressMultiple(t *testing.T) {
	ce := expr.Express("Age", 18, "Name", "John")
	if len(ce) != 1 {
		t.Fatalf("Expected 1 expression, got %d", len(ce))
	}
}

// Happy Path 2
func TestUtilsExpressMapMultiple(t *testing.T) {
	ce := expr.Express(map[string]any{
		"Age": 18,
		"Name": "John",
	})
	if len(ce) != 2 {
		t.Fatalf("Expected 2 expressions, got %d", len(ce))
	}
}

// Unhappy Path 1
func TestUtilsExpressMissingValue(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic when providing key without value")
		}
	}()
	expr.Express("Age")
}

// Unhappy Path 2
func TestUtilsExpressInvalidKeyType(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic when providing non-string key in pairs")
		}
	}()
	expr.Express(123, "value")
}
