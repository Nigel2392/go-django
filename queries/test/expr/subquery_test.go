package expr_test

import (
	"context"
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/queries/src/expr"
)

type mockFieldResolver struct {
	ctx context.Context
	expr.FieldResolver
}

func (m *mockFieldResolver) Context() context.Context {
	return m.ctx
}

// SQL Generation 1
func TestSubqueryOuterRefSQLGen1(t *testing.T) {
	parentInfo := getTestInfo()
	
	// Create a subquery context pointing to parentInfo
	ctx := expr.AddParentSubqueryContext(context.Background(), parentInfo)
	
	// Create a subquery info wrapping the parent resolver but with the new context
	subInfo := getTestInfo()
	subInfo.Resolver = &mockFieldResolver{
		ctx:           ctx,
		FieldResolver: parentInfo.Resolver,
	}

	ref := expr.OuterRef("Age")
	resolved := ref.Resolve(subInfo)
	
	var sb strings.Builder
	args := resolved.SQL(&sb)
	sql := sb.String()

	if !strings.Contains(sql, fixSQL(parentInfo, "`test_model`.`age`")) {
		t.Errorf("Expected outer ref to resolve Age field, got: %s", sql)
	}
	if len(args) != 0 {
		t.Errorf("Expected 0 args, got %d", len(args))
	}
}

// SQL Generation 2
func TestSubqueryOuterRefSQLGen2(t *testing.T) {
	parentInfo := getTestInfo()
	ctx := expr.AddParentSubqueryContext(context.Background(), parentInfo)
	subInfo := getTestInfo()
	subInfo.Resolver = &mockFieldResolver{
		ctx:           ctx,
		FieldResolver: parentInfo.Resolver,
	}

	// Alias resolving
	ref := expr.OuterRef("alias.Name")
	resolved := ref.Resolve(subInfo)
	
	var sb strings.Builder
	resolved.SQL(&sb)
	sql := sb.String()

	if !strings.Contains(sql, fixSQL(parentInfo, "`alias`.`name`")) {
		t.Errorf("Expected outer ref to resolve Name field with alias, got: %s", sql)
	}
}

// Happy Path 1
func TestSubqueryContextRetrieval(t *testing.T) {
	info := getTestInfo()
	ctx := expr.AddParentSubqueryContext(context.Background(), info)
	parent, ok := expr.ParentFromSubqueryContext(ctx)
	if !ok || parent != info {
		t.Errorf("Expected original info to be returned")
	}
}

// Happy Path 2
func TestSubqueryMakeSubqueryContext(t *testing.T) {
	ctx := context.Background()
	if expr.IsSubqueryContext(ctx) {
		t.Errorf("Background shouldn't be subquery context")
	}
	ctx = expr.MakeSubqueryContext(ctx)
	if !expr.IsSubqueryContext(ctx) {
		t.Errorf("Expected context to be subquery context")
	}
}

// Unhappy Path 1
func TestSubqueryOuterRefUnresolved(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic when resolving OuterRef without parent context")
		}
	}()
	info := getTestInfo()
	
	// No parent context added to info.Resolver.Context()
	ref := expr.OuterRef("Age")
	ref.Resolve(info) 
}

// Unhappy Path 2
func TestSubqueryOuterRefUnknownField(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic when resolving unknown OuterRef field")
		}
	}()
	parentInfo := getTestInfo()
	ctx := expr.AddParentSubqueryContext(context.Background(), parentInfo)
	subInfo := getTestInfo()
	subInfo.Resolver = &mockFieldResolver{
		ctx:           ctx,
		FieldResolver: parentInfo.Resolver,
	}

	ref := expr.OuterRef("UnknownField")
	ref.Resolve(subInfo)
}
