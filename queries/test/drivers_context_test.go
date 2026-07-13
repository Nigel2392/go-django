package queries_test

import (
	"context"
	"testing"
	"time"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

func TestQueryFlagString(t *testing.T) {
	if s := drivers.Q_QUERY.String(); s != "QUERY" {
		t.Errorf("Expected 'QUERY', got %q", s)
	}

	multi := drivers.Q_QUERY | drivers.Q_EXEC
	// It should sort the string parts alphabetically -> "EXEC|QUERY"
	if s := multi.String(); s != "EXEC|QUERY" {
		t.Errorf("Expected 'EXEC|QUERY', got %q", s)
	}

	// Test unknown flag
	unknown := drivers.QueryFlag(1 << 15) // Not defined
	if s := unknown.String(); s != "UNKNOWN" {
		t.Errorf("Expected 'UNKNOWN', got %q", s)
	}
}

func TestContextQueryInfoEdgeCases(t *testing.T) {
	_, qi := drivers.ContextWithQueryInfo(context.Background())

	// no queries executed
	if qi.TotalExecutionTime() != 0 {
		t.Errorf("Expected 0 TotalExecutionTime, got %v", qi.TotalExecutionTime())
	}
	if qi.TotalTime() != 0 {
		t.Errorf("Expected 0 TotalTime, got %v", qi.TotalTime())
	}
	if qi.AverageTime() != 0 {
		t.Errorf("Expected 0 AverageTime, got %v", qi.AverageTime())
	}
	if qi.Slowest() != nil {
		t.Errorf("Expected nil Slowest(), got %v", qi.Slowest())
	}
}

func TestContextQueryExecWithoutInfo(t *testing.T) {
	// Context without QueryInformation attached
	ctx := context.Background()

	fn := func(ctx context.Context, query string, args ...any) (string, error) {
		return "success", nil
	}

	res, err := drivers.ContextQueryExec(ctx, "test_driver", "SELECT 1", nil, drivers.Q_QUERY, fn)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if res != "success" {
		t.Errorf("Expected 'success', got %v", res)
	}
}

func TestContextQueryExecAndStats(t *testing.T) {
	ctx, qi := drivers.ContextWithQueryInfo(context.Background())

	fn := func(ctx context.Context, query string, args ...any) (int, error) {
		sleepDur := args[0].(time.Duration)
		time.Sleep(sleepDur)
		return 1, nil
	}

	// First query: fast (10ms)
	_, _ = drivers.ContextQueryExec(ctx, "mock", "SELECT fast", []any{10 * time.Millisecond}, drivers.Q_QUERY, fn)

	// Second query: slow (50ms)
	_, _ = drivers.ContextQueryExec(ctx, "mock", "SELECT slow", []any{50 * time.Millisecond}, drivers.Q_EXEC, fn)

	// Third query: medium (20ms)
	_, _ = drivers.ContextQueryExec(ctx, "mock", "SELECT medium", []any{20 * time.Millisecond}, drivers.Q_QUERYROW, fn)

	if len(qi.Queries) != 3 {
		t.Fatalf("Expected 3 queries tracked, got %d", len(qi.Queries))
	}

	slowest := qi.Slowest()
	if slowest == nil || slowest.Query != "SELECT slow" {
		t.Errorf("Expected 'SELECT slow' to be slowest, got %v", slowest.Query)
	}

	totalTime := qi.TotalTime()
	if totalTime < 80*time.Millisecond {
		t.Errorf("Expected TotalTime >= 80ms, got %v", totalTime)
	}

	avgTime := qi.AverageTime()
	expectedAvg := totalTime / 3
	if avgTime != expectedAvg {
		t.Errorf("Expected AverageTime %v, got %v", expectedAvg, avgTime)
	}

	// Check TotalExecutionTime()
	// It's calculated as latest.Start.Add(latest.TimeTaken).Sub(qi.Start)
	totalExecTime := qi.TotalExecutionTime()
	if totalExecTime < totalTime {
		t.Errorf("Expected TotalExecutionTime >= TotalTime, got ExecTime=%v, TotalTime=%v", totalExecTime, totalTime)
	}

	// Verify that the flags were correctly recorded
	if qi.Queries[0].Flags != drivers.Q_QUERY {
		t.Errorf("Expected Q_QUERY flag on first query, got %v", qi.Queries[0].Flags)
	}
	if qi.Queries[1].Flags != drivers.Q_EXEC {
		t.Errorf("Expected Q_EXEC flag on second query, got %v", qi.Queries[1].Flags)
	}
	if qi.Queries[2].Flags != drivers.Q_QUERYROW {
		t.Errorf("Expected Q_QUERYROW flag on third query, got %v", qi.Queries[2].Flags)
	}
}

func TestQueryExplainUnknownDriverPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic for unknown driver")
		}
	}()

	q := &drivers.Query{
		Driver: "this_driver_does_not_exist_123",
		Query:  "SELECT 1",
	}
	_, _ = q.Explain(context.Background(), nil)
}

func TestContextQueryExecRealDB(t *testing.T) {
	ctx, qi := drivers.ContextWithQueryInfo(context.Background())

	var user = &User{Name: "TestRealDB User"}
	if err := queries.CreateObject(user); err != nil {
		t.Fatalf("Failed to create *User: %v", err)
	}

	var count int
	err := queries.GetQuerySet(&Todo{}).WithContext(ctx).Row(
		`SELECT COUNT(*) FROM TABLE(SELF)`,
	).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to execute SELECT: %v", err)
	}

	_, err = queries.GetQuerySet(&Todo{}).WithContext(ctx).Exec(
		`INSERT INTO TABLE(SELF) (title, done, user_id) VALUES (?[1], ?[2], ?[3])`,
		"RealDB Title", false, user.ID,
	)
	if err != nil {
		t.Fatalf("Failed to execute INSERT: %v", err)
	}

	_, err = queries.GetQuerySet[attrs.Definer](&Todo{}).WithContext(ctx).Filter("User", user).All()
	if err != nil {
		t.Fatalf("Failed to execute All: %v", err)
	}

	if len(qi.Queries) < 3 {
		t.Fatalf("Expected at least 3 queries tracked, got %d", len(qi.Queries))
	}

	slowest := qi.Slowest()
	if slowest == nil || slowest.Query == "" {
		t.Errorf("Expected a valid slowest query, got %v", slowest)
	}

	totalTime := qi.TotalTime()
	if totalTime < 0 {
		t.Errorf("Expected TotalTime >= 0")
	}

	avgTime := qi.AverageTime()
	expectedAvg := totalTime / time.Duration(len(qi.Queries))
	if avgTime != expectedAvg {
		t.Errorf("Expected AverageTime %v, got %v", expectedAvg, avgTime)
	}

	totalExecTime := qi.TotalExecutionTime()
	if totalExecTime < totalTime {
		t.Errorf("Expected TotalExecutionTime >= TotalTime, got ExecTime=%v, TotalTime=%v", totalExecTime, totalTime)
	}

	// Verify that the flags were correctly recorded natively
	if qi.Queries[0].Flags != drivers.Q_QUERYROW {
		t.Errorf("Expected Q_QUERYROW flag on first query, got %v", qi.Queries[0].Flags)
	}
	if qi.Queries[1].Flags != drivers.Q_EXEC {
		t.Errorf("Expected Q_EXEC flag on second query, got %v", qi.Queries[1].Flags)
	}
	if qi.Queries[2].Flags != drivers.Q_QUERY {
		t.Errorf("Expected Q_QUERY flag on third query, got %v", qi.Queries[2].Flags)
	}

	// Clean up
	queries.GetQuerySet[attrs.Definer](&Todo{}).Filter("User", user).Delete()
	queries.GetQuerySet[attrs.Definer](&User{}).Filter("ID", user.ID).Delete()
}
