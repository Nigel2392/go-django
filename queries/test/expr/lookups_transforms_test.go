package expr_test

import (
	"strings"
	"testing"
	"github.com/Nigel2392/go-django/queries/src/expr"
)

// SQL Generation 1
func TestLookupsTransformsUpperSQL(t *testing.T) {
	info := getTestInfo()
	q := expr.Q("Name__upper__exact", "JOHN")
	resolved := q.Resolve(info)
	var sb strings.Builder
	resolved.SQL(&sb)
	if !strings.Contains(sb.String(), fixSQL(info, "UPPER(`test_model`.`name`) = ?")) {
		t.Errorf("Unexpected upper transform SQL: %s", sb.String())
	}
}

// SQL Generation 2
func TestLookupsTransformsLowerSQL(t *testing.T) {
	info := getTestInfo()
	q := expr.Q("Name__lower__contains", "john")
	resolved := q.Resolve(info)
	var sb strings.Builder
	resolved.SQL(&sb)
	if !strings.Contains(strings.ToUpper(sb.String()), "LIKE") {
		t.Errorf("Unexpected lower transform SQL: %s", sb.String())
	}
}

// Happy Path 1
func TestLookupsTransformsUpperImplicitExact(t *testing.T) {
	info := getTestInfo()
	// Without exact, it defaults to exact
	q := expr.Q("Name__upper", "JOHN")
	resolved := q.Resolve(info)
	var sb strings.Builder
	resolved.SQL(&sb)
	if !strings.Contains(sb.String(), fixSQL(info, "UPPER(`test_model`.`name`) = ?")) {
		t.Errorf("Unexpected upper transform implicit exact SQL: %s", sb.String())
	}
}

// Happy Path 2
func TestLookupsTransformsDoubleTransform(t *testing.T) {
	info := getTestInfo()
	// lower__upper is silly but should generate UPPER(LOWER(col))
	q := expr.Q("Name__lower__upper", "JOHN")
	resolved := q.Resolve(info)
	var sb strings.Builder
	resolved.SQL(&sb)
	if !strings.Contains(sb.String(), fixSQL(info, "UPPER(LOWER(`test_model`.`name`)) = ?")) {
		t.Errorf("Unexpected chained transforms SQL: %s", sb.String())
	}
}

// Unhappy Path 1
func TestLookupsTransformsUnknownTransform(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic on unknown transform")
		}
	}()
	info := getTestInfo()
	q := expr.Q("Name__unknown_transform__exact", "JOHN")
	q.Resolve(info)
}

// Unhappy Path 2
func TestLookupsTransformsInvalidRegistration(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic on resolving nil transform")
		}
	}()
	expr.RegisterTransforms(&expr.BaseTransform{
		Identifier: "bad_transform",
		// missing Transform func will panic during Resolve
	})
	info := getTestInfo()
	q := expr.Q("Name__bad_transform", "JOHN")
	q.Resolve(info)
}
