package queries_test

import (
	"testing"
	"time"

	queries "github.com/Nigel2392/go-django/queries/src"
)

func TestQueryInformationArgsConvertsZeroTimeToNil(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	zero := time.Time{}
	nonZeroPtr := &now
	zeroPtr := &zero

	q := &queries.QueryInformation{
		Params: []any{
			"hello",
			zero,
			now,
			zeroPtr,
			nonZeroPtr,
		},
	}

	args := q.Args()

	if args[0] != "hello" {
		t.Fatalf("expected first arg to remain unchanged, got %v", args[0])
	}
	if args[1] != nil {
		t.Fatalf("expected zero time.Time to become nil, got %T (%v)", args[1], args[1])
	}

	gotTime, ok := args[2].(time.Time)
	if !ok || !gotTime.Equal(now) {
		t.Fatalf("expected non-zero time.Time to remain unchanged, got %T (%v)", args[2], args[2])
	}
	if args[3] != nil {
		t.Fatalf("expected zero *time.Time to become nil, got %T (%v)", args[3], args[3])
	}

	gotPtr, ok := args[4].(*time.Time)
	if !ok || gotPtr == nil || !gotPtr.Equal(now) {
		t.Fatalf("expected non-zero *time.Time to remain unchanged, got %T (%v)", args[4], args[4])
	}
}
