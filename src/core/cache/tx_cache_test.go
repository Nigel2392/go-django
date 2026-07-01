package cache_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Nigel2392/go-django/src/core/cache"
)

func TestMemoryCacheTransactions(t *testing.T) {
	ctx := context.Background()
	var errMock = errors.New("mock error for rollback")

	// We'll use a string-typed generic cache for easier assertions in the tests.
	setupCache := func() *cache.MemoryCache[string] {
		c := cache.NewGenericMemoryCache[string]()
		c.Run(1 * time.Second)
		_ = c.Set(ctx, "existing_key", "old_value", 5*time.Minute)
		_ = c.Set(ctx, "to_be_deleted", "will_be_gone", 5*time.Minute)
		return c
	}

	t.Run("Rollback on Error", func(t *testing.T) {
		c := setupCache()

		err := c.RunInTx(ctx, func(ctx context.Context, tx cache.TypedTransaction[string]) error {
			_ = tx.Set(ctx, "existing_key", "new_value", 0)
			_ = tx.Set(ctx, "new_key", "new_value", 0)
			_ = tx.Delete(ctx, "to_be_deleted")
			return errMock // Returning an error should discard all changes
		})

		if !errors.Is(err, errMock) {
			t.Fatalf("expected errMock, got %v", err)
		}

		// Assert main cache is completely untouched
		val, _ := c.Get(ctx, "existing_key")
		if val != "old_value" {
			t.Fatalf("expected 'old_value', got %v", val)
		}

		if c.Has(ctx, "new_key") {
			t.Fatalf("new_key should not exist after rollback")
		}

		if !c.Has(ctx, "to_be_deleted") {
			t.Fatalf("to_be_deleted should still exist after rollback")
		}
	})

	t.Run("Successful Commit with Updates and Deletes", func(t *testing.T) {
		c := setupCache()

		err := c.RunInTx(ctx, func(ctx context.Context, tx cache.TypedTransaction[string]) error {
			// 1. Read isolation
			val, _ := tx.Get(ctx, "existing_key")
			if val != "old_value" {
				t.Fatalf("expected to read 'old_value' inside tx, got %v", val)
			}

			// 2. Write isolation
			_ = tx.Set(ctx, "existing_key", "new_value", 0)
			val, _ = tx.Get(ctx, "existing_key")
			if val != "new_value" {
				t.Fatalf("expected to read 'new_value' immediately after tx.Set, got %v", val)
			}

			// 3. Deletion
			_ = tx.Delete(ctx, "to_be_deleted")

			return nil // Commit
		})

		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}

		// Assert main cache applies diffs correctly
		val, _ := c.Get(ctx, "existing_key")
		if val != "new_value" {
			t.Fatalf("expected 'new_value', got %v", val)
		}

		if c.Has(ctx, "to_be_deleted") {
			t.Fatalf("to_be_deleted should be gone from main cache")
		}
	})

	t.Run("No Lost Updates (Concurrent modifications)", func(t *testing.T) {
		c := setupCache()

		err := c.RunInTx(ctx, func(ctx context.Context, tx cache.TypedTransaction[string]) error {
			// Change something in the tx
			_ = tx.Set(ctx, "existing_key", "changed_in_tx", 0)

			// SIMULATE CONCURRENCY:
			// While the tx is running, another goroutine modifies the main cache.
			// Because we unlock the mutex during execution, this is perfectly valid.
			_ = c.Set(ctx, "concurrent_key", "changed_outside_tx", 0)

			return nil
		})

		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}

		// If the "Lost Update" bug was present, committing the Tx would have
		// overwritten the concurrent write. Let's ensure BOTH changes exist.
		val1, _ := c.Get(ctx, "existing_key")
		if val1 != "changed_in_tx" {
			t.Fatalf("expected tx change, got %v", val1)
		}

		val2, _ := c.Get(ctx, "concurrent_key")
		if val2 != "changed_outside_tx" {
			t.Fatalf("concurrent change was lost! expected 'changed_outside_tx', got %v", val2)
		}
	})

	t.Run("Clear inside Tx", func(t *testing.T) {
		c := setupCache()

		err := c.RunInTx(ctx, func(ctx context.Context, tx cache.TypedTransaction[string]) error {
			_ = tx.Clear(ctx)
			_ = tx.Set(ctx, "post_clear_key", "value", 0)
			return nil
		})

		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}

		if c.Has(ctx, "existing_key") || c.Has(ctx, "to_be_deleted") {
			t.Fatalf("cache should have been completely cleared")
		}

		if !c.Has(ctx, "post_clear_key") {
			t.Fatalf("keys added after clear() should exist")
		}
	})

	t.Run("Expiration inside Tx", func(t *testing.T) {
		c := cache.NewGenericMemoryCache[string]()
		c.Run(1 * time.Second)

		// Set a key that expires extremely fast
		_ = c.Set(ctx, "fast_expire", "val", 50*time.Millisecond)

		// Wait for it to expire
		time.Sleep(60 * time.Millisecond)

		err := c.RunInTx(ctx, func(ctx context.Context, tx cache.TypedTransaction[string]) error {
			// Even though it might have been snapshotted, expired() should catch it
			_, err := tx.Get(ctx, "fast_expire")
			if err == nil {
				t.Fatalf("expected ErrItemNotFound for expired key inside tx")
			}

			if tx.Has(ctx, "fast_expire") {
				t.Fatalf("Has() should return false for expired key inside tx")
			}
			return nil
		})

		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})
}

func TestMemoryCacheTransactionEdgeCases(t *testing.T) {
	ctx := context.Background()

	setupCache := func() *cache.MemoryCache[string] {
		c := cache.NewGenericMemoryCache[string]()
		c.Run(1 * time.Second)
		_ = c.Set(ctx, "existing_key", "old_value", 5*time.Minute)
		return c
	}

	t.Run("Concurrent Clear Bug", func(t *testing.T) {
		c := setupCache()

		err := c.RunInTx(ctx, func(ctx context.Context, tx cache.TypedTransaction[string]) error {
			// 1. Trigger the nuclear option
			_ = tx.Clear(ctx)

			// 2. SIMULATE CONCURRENCY:
			// Another goroutine adds a brand new key to the main cache
			// while our transaction is still running.
			_ = c.Set(ctx, "sneaky_concurrent_key", "im_in", 5*time.Minute)

			// 3. Add a post-clear key inside the tx to ensure it survives
			_ = tx.Set(ctx, "post_clear_key", "valid", 5*time.Minute)

			return nil
		})

		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}

		// Assertions:
		// The sneaky key should be GONE because Clear() acts as a strict barrier.
		if c.Has(ctx, "sneaky_concurrent_key") {
			t.Fatalf("concurrent key survived a tx.Clear(), the clear was not absolute")
		}

		// The existing key should be gone.
		if c.Has(ctx, "existing_key") {
			t.Fatalf("existing key survived tx.Clear()")
		}

		// The key added explicitly AFTER the clear inside the tx should exist.
		val, _ := c.Get(ctx, "post_clear_key")
		if val != "valid" {
			t.Fatalf("expected post_clear_key to equal 'valid', got %v", val)
		}
	})

	t.Run("Zombie Transaction (Context Cancellation)", func(t *testing.T) {
		c := setupCache()

		// Create a context that expires in 10 milliseconds
		cancelCtx, cancel := context.WithTimeout(ctx, 10*time.Millisecond)
		defer cancel()

		err := c.RunInTx(cancelCtx, func(ctx context.Context, tx cache.TypedTransaction[string]) error {
			// Change state inside the transaction
			_ = tx.Set(cancelCtx, "existing_key", "zombie_value", 0)

			// Simulate heavy work that outlives the context timeout
			time.Sleep(30 * time.Millisecond)

			// The function finishes "successfully" without returning its own error
			return nil
		})

		// The transaction MUST intercept the context timeout and refuse to commit
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected context.DeadlineExceeded, got %v", err)
		}

		// The main cache must remain completely untouched
		val, _ := c.Get(ctx, "existing_key")
		if val != "old_value" {
			t.Fatalf("zombie transaction committed! expected 'old_value', got %v", val)
		}
	})

	t.Run("Set After Delete Anomaly (State Bouncing)", func(t *testing.T) {
		c := setupCache()

		err := c.RunInTx(ctx, func(ctx context.Context, tx cache.TypedTransaction[string]) error {
			// Scenario A: Set -> Delete -> Set
			_ = tx.Set(ctx, "bounce_key", "first_val", 0)
			_ = tx.Delete(ctx, "bounce_key")
			_ = tx.Set(ctx, "bounce_key", "final_val", 0)

			// Scenario B: Delete -> Set -> Delete (on an existing key)
			_ = tx.Delete(ctx, "existing_key")
			_ = tx.Set(ctx, "existing_key", "nevermind", 0)
			_ = tx.Delete(ctx, "existing_key")

			return nil
		})

		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}

		// Assert Scenario A: The final Set should win.
		val, err := c.Get(ctx, "bounce_key")
		if err != nil || val != "final_val" {
			t.Fatalf("bounce_key state corrupted. expected 'final_val', got %v", val)
		}

		// Assert Scenario B: The final Delete should win.
		if c.Has(ctx, "existing_key") {
			t.Fatalf("existing_key should have been ultimately deleted")
		}
	})
}

func TestMemoryCacheTransactionCounters(t *testing.T) {
	ctx := context.Background()

	setupTxCache := func() *cache.MemoryCache[any] {
		c := cache.NewGenericMemoryCache[any]()
		c.Run(1 * time.Second)
		_ = c.Set(ctx, "existing_counter", int64(50), 5*time.Minute)
		return c
	}

	t.Run("Commit Increments and Decrements", func(t *testing.T) {
		c := setupTxCache()

		err := c.RunInTx(ctx, func(ctx context.Context, tx cache.TypedTransaction[any]) error {
			if !tx.InTransaction() {
				t.Fatalf("expected InTransaction() to return true")
			}

			val, err := tx.Increment(ctx, "existing_counter", 10)
			if err != nil {
				return err
			}
			if val != 60 {
				t.Fatalf("expected 60 inside tx, got %v", val)
			}

			val, err = tx.Decrement(ctx, "existing_counter", 5)
			if err != nil {
				return err
			}
			if val != 55 {
				t.Fatalf("expected 55 inside tx, got %v", val)
			}

			val, err = tx.Increment(ctx, "new_tx_counter", 100)
			if err != nil {
				return err
			}
			if val != 100 {
				t.Fatalf("expected 100 inside tx, got %v", val)
			}

			return nil // Commit
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Assert Main Cache Updated Correctly
		val, _ := c.Get(ctx, "existing_counter")
		if val.(int64) != 55 {
			t.Fatalf("expected 55 in main cache, got %v", val)
		}

		val, _ = c.Get(ctx, "new_tx_counter")
		if val.(int64) != 100 {
			t.Fatalf("expected 100 in main cache, got %v", val)
		}
	})

	t.Run("Rollback Increments", func(t *testing.T) {
		c := setupTxCache()
		errMock := errors.New("mock rollback")

		err := c.RunInTx(ctx, func(ctx context.Context, tx cache.TypedTransaction[any]) error {
			_, _ = tx.Increment(ctx, "existing_counter", 100)    // Should be 150
			_, _ = tx.Decrement(ctx, "new_rollback_counter", 50) // Should be -50
			return errMock                                       // Discard everything
		})

		if !errors.Is(err, errMock) {
			t.Fatalf("expected rollback error")
		}

		// Assert Main Cache is untouched
		val, _ := c.Get(ctx, "existing_counter")
		if val.(int64) != 50 {
			t.Fatalf("expected existing_counter to remain 50, got %v", val)
		}

		if c.Has(ctx, "new_rollback_counter") {
			t.Fatalf("new_rollback_counter should not exist after rollback")
		}
	})

	t.Run("State Bouncing (Delete then Increment)", func(t *testing.T) {
		c := setupTxCache()

		err := c.RunInTx(ctx, func(ctx context.Context, tx cache.TypedTransaction[any]) error {
			// Delete the key, marking it as 'deleted: true' in tx.state
			_ = tx.Delete(ctx, "existing_counter")

			// Incrementing it now should act like a brand new initialization
			val, err := tx.Increment(ctx, "existing_counter", 10)
			if err != nil {
				return err
			}
			if val != 10 {
				t.Fatalf("expected 10 after delete-then-increment, got %v", val)
			}

			return nil // Commit
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		val, _ := c.CounterValue(ctx, "existing_counter")
		if val != 10 {
			t.Fatalf("expected state bounce to yield 10, got %v", val)
		}
	})
}

func setupGlobalCache() {
	// Reset the global default cache to ensure test isolation
	c := cache.NewMemoryCache(1 * time.Second)
	cache.SetDefault(c)
}

func TestContextTransactionHelpers(t *testing.T) {
	ctx := context.Background()

	t.Run("Empty Context Returns False", func(t *testing.T) {
		_, ok := cache.TransactionFromContext(ctx)
		if ok {
			t.Fatalf("expected no transaction in empty context")
		}
	})

	t.Run("Context With Transaction", func(t *testing.T) {
		setupGlobalCache()

		// Start a dummy transaction just to get the tx object
		_ = cache.RunInTx(ctx, func(ctx context.Context, tx cache.Transaction) error {
			txCtx := cache.ContextWithTransaction(ctx, tx)

			retrievedTx, ok := cache.TransactionFromContext(txCtx)
			if !ok {
				t.Fatalf("failed to retrieve transaction from context")
			}

			if retrievedTx == nil {
				t.Fatalf("retrieved transaction is nil")
			}
			return nil
		})
	})
}

func TestGlobalWrappersWithContextPropagation(t *testing.T) {
	ctx := context.Background()

	t.Run("Successful Commit via Global Wrappers", func(t *testing.T) {
		setupGlobalCache()

		err := cache.RunInTx(ctx, func(ctx context.Context, tx cache.Transaction) error {
			// 1. Inject the transaction into a new context
			txCtx := cache.ContextWithTransaction(ctx, tx)

			// 2. Perform operations using the GLOBAL package methods, passing the injected context
			_ = cache.Set(txCtx, "implicit_key", "implicit_val", 5*time.Minute)
			_, _ = cache.Increment(txCtx, "implicit_counter", 10)

			// 3. Verify they exist inside the transaction context
			if !cache.Has(txCtx, "implicit_key") {
				t.Fatalf("expected global wrapper to route Set to transaction")
			}

			// 4. Verify they DO NOT exist in the main cache yet
			// We test this by using the original 'ctx' which lacks the transaction
			if cache.Has(ctx, "implicit_key") {
				t.Fatalf("transaction isolation breached: key found in main cache before commit")
			}

			return nil // Commit
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// 5. Verify data is flushed to main cache after commit
		val, _ := cache.Get(ctx, "implicit_key")
		if val != "implicit_val" {
			t.Fatalf("expected 'implicit_val' to be flushed to main cache, got %v", val)
		}

		counterVal, _ := cache.Get(ctx, "implicit_counter") // Verify prefix logic held up
		if counterVal.(int64) != 10 {
			t.Fatalf("expected counter to equal 10 in main cache, got %v", counterVal)
		}
	})

	t.Run("Rollback via Global Wrappers", func(t *testing.T) {
		setupGlobalCache()
		errMock := errors.New("abort transaction")

		err := cache.RunInTx(ctx, func(ctx context.Context, tx cache.Transaction) error {
			txCtx := cache.ContextWithTransaction(ctx, tx)

			_ = cache.Set(txCtx, "rollback_key", "val", 0)
			_, _ = cache.Decrement(txCtx, "rollback_counter", 50)

			// Verify they exist in the tx state
			val, _ := cache.Get(txCtx, "rollback_key")
			if val != "val" {
				t.Fatalf("expected to read 'val' inside transaction")
			}

			return errMock // Rollback
		})

		if !errors.Is(err, errMock) {
			t.Fatalf("expected errMock")
		}

		// Verify main cache is untouched
		if cache.Has(ctx, "rollback_key") {
			t.Fatalf("rollback failed: key exists in main cache")
		}
	})

	t.Run("Clear via Global Wrapper in Tx", func(t *testing.T) {
		setupGlobalCache()
		_ = cache.Set(ctx, "pre_existing_key", "keep_me", 0)

		err := cache.RunInTx(ctx, func(ctx context.Context, tx cache.Transaction) error {
			txCtx := cache.ContextWithTransaction(ctx, tx)

			// Call global clear with transaction context
			_ = cache.Clear(txCtx)

			// Ensure it reflects in the transaction
			if cache.Has(txCtx, "pre_existing_key") {
				t.Fatalf("expected pre_existing_key to be cleared inside tx")
			}

			return nil // Commit
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cache.Has(ctx, "pre_existing_key") {
			t.Fatalf("expected main cache to be cleared after commit")
		}
	})

	t.Run("transactionOrDefault Fallback", func(t *testing.T) {
		setupGlobalCache()

		// Using a plain context (no transaction injected)
		// Should route directly to Default() cache without panicking
		_ = cache.Set(ctx, "direct_key", "direct_val", 0)
		_, _ = cache.Increment(ctx, "direct_counter", 5)

		val, _ := cache.Get(ctx, "direct_key")
		if val != "direct_val" {
			t.Fatalf("expected 'direct_val', got %v", val)
		}
	})
}
