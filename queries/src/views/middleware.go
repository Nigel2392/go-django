package views

import (
	"context"
	"net/http"
	"time"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/mux"
)

type TransactionOptions struct {
	Deadline time.Duration
}

type transactionMiddleware struct {
	next mux.Handler
	opts TransactionOptions
}

func (m *transactionMiddleware) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var ctx, tx, err = queries.StartTransaction(req.Context())
	if err != nil {
		logger.Errorf("Failed to start transaction: %v", err)
		except.Fail(500, "Internal Server Error")
		return
	}

	if m.opts.Deadline > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, m.opts.Deadline)
		defer cancel()
	}

	req = req.WithContext(ctx)
	ctx = req.Context()
	if tx == nil {
		logger.Warnf("No transaction started for request %s", req.URL.Path)
		m.next.ServeHTTP(w, req)
		return
	}

	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			logger.Errorf("Failed to rollback transaction: %v", err)
		}
	}()

	m.next.ServeHTTP(w, req)

	if err := tx.Commit(ctx); err != nil {
		logger.Errorf("Failed to commit transaction: %v", err)
		except.Fail(500, "Internal Server Error")
		return
	}
}

func TransactionMiddleware(opts TransactionOptions) func(next mux.Handler) mux.Handler {
	return func(next mux.Handler) mux.Handler {
		return &transactionMiddleware{next: next}
	}
}
