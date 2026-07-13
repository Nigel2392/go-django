package expr_test

import (
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/queries/src/expr"
)

// SQL Generation 1
func TestLookupsIstartswithSQL(t *testing.T) {
	info := getTestInfo()
	q := expr.Q("Name__istartswith", "John")
	resolved := q.Resolve(info)
	var sb strings.Builder
	args := resolved.SQL(&sb)
	if !strings.Contains(sb.String(), "LIKE") {
		t.Errorf("Unexpected istartswith SQL: %s", sb.String())
	}
	if len(args) != 1 || args[0] != "John%" { // Assuming string formats handle %
		t.Errorf("Unexpected args for istartswith: %v", args)
	}
}

// SQL Generation 2
func TestLookupsLteSQL(t *testing.T) {
	info := getTestInfo()
	q := expr.Q("Age__lte", 65)
	resolved := q.Resolve(info)
	var sb strings.Builder
	resolved.SQL(&sb)
	if !strings.Contains(sb.String(), fixSQL(info, "`test_model`.`age` <= ?")) {
		t.Errorf("Unexpected lte SQL: %s", sb.String())
	}
}

// Happy Path 1
func TestLookupsNotExactResolve(t *testing.T) {
	info := getTestInfo()
	q := expr.Q("Name__not", "John")
	resolved := q.Resolve(info)
	var sb strings.Builder
	resolved.SQL(&sb)
	if !strings.Contains(sb.String(), fixSQL(info, "`test_model`.`name` != ?")) {
		t.Errorf("Unexpected NOT SQL: %s", sb.String())
	}
}

// Happy Path 2
func TestLookupsBitwiseAndResolve(t *testing.T) {
	info := getTestInfo()
	q := expr.Q("Score__bitand", 1)
	resolved := q.Resolve(info)
	var sb strings.Builder
	resolved.SQL(&sb)
	if !strings.Contains(sb.String(), fixSQL(info, "`test_model`.`score` & ?")) {
		t.Errorf("Unexpected bitand SQL: %s", sb.String())
	}
}

// Unhappy Path 1
func TestLookupsMalformedQ(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic on malformed lookup field")
		}
	}()
	info := getTestInfo()
	q := expr.Q("__Age", 18) // invalid start
	q.Resolve(info)
}

// Unhappy Path 2
func TestLookupsMissingValue(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic on missing value in Q")
		}
	}()
	// expr.Q expects pairs. This will panic during resolving if args are missing
	q := expr.Q("Age__exact")
	q.Resolve(getTestInfo())
}
