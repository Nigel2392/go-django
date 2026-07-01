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

		err := c.RunInTx(ctx, func(tx cache.TypedCache[string]) error {
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

		err := c.RunInTx(ctx, func(tx cache.TypedCache[string]) error {
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

		err := c.RunInTx(ctx, func(tx cache.TypedCache[string]) error {
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

		err := c.RunInTx(ctx, func(tx cache.TypedCache[string]) error {
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

		err := c.RunInTx(ctx, func(tx cache.TypedCache[string]) error {
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

		err := c.RunInTx(ctx, func(tx cache.TypedCache[string]) error {
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

		err := c.RunInTx(cancelCtx, func(tx cache.TypedCache[string]) error {
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

		err := c.RunInTx(ctx, func(tx cache.TypedCache[string]) error {
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
