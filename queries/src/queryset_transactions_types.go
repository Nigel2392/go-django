package queries

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/src/core/logger"
)

type nullTransaction struct {
	drivers.DB
}

func NullTransction() drivers.Transaction {
	return &nullTransaction{DB: nil}
}

func (n *nullTransaction) Rollback(context.Context) error {
	return nil
}

func (n *nullTransaction) Commit(context.Context) error {
	return nil
}

type dbSpecificTransaction struct {
	drivers.Transaction
	dbName string
}

func (c *dbSpecificTransaction) DatabaseName() string {
	return c.dbName
}

type wrappedTransaction struct {
	drivers.Transaction
	compiler *genericQueryBuilder
}

func (w *wrappedTransaction) Rollback(ctx context.Context) error {
	if !w.compiler.InTransaction() {
		return errors.NoTransaction
	}
	if w.compiler != nil {
		w.compiler.transaction = nil
	}
	var err = w.Transaction.Rollback(ctx)
	if errors.Is(err, sql.ErrTxDone) {
		return nil
	}
	logger.Warnf("Rolling back transaction for %s (%v)", w.compiler.DatabaseName(), err)
	if err != nil {
		return errors.RollbackFailed.WithCause(fmt.Errorf(
			"failed to rollback transaction for %s: %w",
			w.compiler.DatabaseName(), err,
		))
	}
	return nil
}

func (w *wrappedTransaction) Commit(ctx context.Context) error {
	if !w.compiler.InTransaction() {
		return errors.NoTransaction
	}
	if w.compiler != nil {
		w.compiler.transaction = nil
	}
	var err = w.Transaction.Commit(ctx)
	if err != nil {
		return errors.CommitFailed.WithCause(fmt.Errorf(
			"failed to commit transaction for %s: %w",
			w.compiler.DatabaseName(), err,
		))
	}
	return nil
}
