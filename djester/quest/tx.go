//go:build test
// +build test

package quest

import (
	"context"

	"github.com/Nigel2392/go-django/djester"
	queries "github.com/Nigel2392/go-django/queries/src"
)

// Start a transaction that will automatically be rolled back on test cleanup.
func StartTransaction[TEST djester.BaseTB](t TEST, database ...string) (context.Context, queries.DatabaseSpecificTransaction) {
	t.Helper()

	ctx, tx, err := queries.StartTransaction(t.Context(), database...)
	if err != nil {
		t.Fatalf("Error during transaction setup: %v", err)
	}

	t.Cleanup(func() {
		tx.Rollback(t.Context())
	})

	return ctx, tx
}
