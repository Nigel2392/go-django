package queries

import (
	"context"
	"fmt"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/logger"
)

type transactionContextKey struct{}

type transactionContextValue struct {
	Transaction  drivers.Transaction
	DatabaseName string
}

func transactionFromContext(ctx context.Context) (tx drivers.Transaction, databaseName string, ok bool) {
	t, ok := ctx.Value(transactionContextKey{}).(*transactionContextValue)
	if !ok {
		return nil, "", false
	}
	return t.Transaction, t.DatabaseName, t.Transaction != nil && !t.Transaction.Finished()
}

func transactionToContext(ctx context.Context, tx drivers.Transaction, dbName string) context.Context {
	if tx == nil {
		panic("transactionToContext: transaction is nil")
	}
	if tx.Finished() {
		return ctx
	}
	return context.WithValue(ctx, transactionContextKey{}, &transactionContextValue{
		Transaction:  tx,
		DatabaseName: dbName,
	})
}

// StartTransaction starts a new transaction for the given database.
//
// If a transaction already exists in the context, it will return a no-op transaction,
// meaning that Rollback and Commit will do nothing, it is assumed that the transaction
// is managed by a higher-level function in the call stack.
//
// If the database name is not provided, it will use the default database name from the compiler.
// If the database name is provided, it will use that database name to start the transaction.
//
// The transaction can not be retrieved from the context if the database name is different, a new
// transaction will be started for the given database name.
//
// The context returned will have the transaction stored in it, so that it can be used later, the
// transaction is stored in the context - a queryset will automatically use the transaction
// from the context if it exists when using [QuerySet.WithContext].
func StartTransaction(ctx context.Context, database ...string) (context.Context, DatabaseSpecificTransaction, error) {
	var (
		databaseName   = getDatabaseName(nil, database...)
		tx, dbName, ok = transactionFromContext(ctx)
		err            error
	)

	// If the context already has a transaction, use it.
	if ok && (dbName == "" || dbName == databaseName) {
		// return a null transaction when there is already a transaction in the context
		// this will make sure that RollBack and Commit are no-ops
		return ctx, &dbSpecificTransaction{&nullTransaction{tx}, databaseName}, nil
	}

	// Otherwise, start a new transaction.
	var compiler = Compiler(databaseName)
	tx, err = compiler.StartTransaction(ctx)
	if err != nil {
		return ctx, nil, errors.FailedStartTransaction.WithCause(fmt.Errorf(
			"failed to start transaction for database %q: %w",
			databaseName, err,
		))
	}

	ctx = transactionToContext(ctx, tx, compiler.DatabaseName())
	return ctx, &dbSpecificTransaction{tx, databaseName}, nil
}

// RunInTransaction runs the given function in a transaction.
//
// The function should return a boolean indicating whether the transaction should be committed or rolled back.
// If the function returns an error, the transaction will be rolled back.
//
// The queryset passed to the function will have the transaction bound to it, so that it can be used
// to execute queries within the transaction.
//
// If the function panics, the transaction will be rolled back and the panic will be recovered.
func RunInTransaction[T attrs.Definer](c context.Context, fn func(ctx context.Context, NewQuerySet ObjectsFunc[T]) (commit bool, err error), database ...string) error {
	var panicFromNewQuerySet error
	var comitted bool

	// If the context already has a transaction, use it.
	var ctx, transaction, err = StartTransaction(c, database...)
	if err != nil {
		return errors.Wrap(err, "RunInTransaction: failed to start transaction")
	}

	var dbName = transaction.DatabaseName()
	// a constructor function to create a new QuerySet with the given model
	// and then bind the transaction to it.
	var newQuerySetFunc = func(model T) *QuerySet[T] {
		var qs = GetQuerySet(model)

		// a transaction cannot be started if the database name is different
		// cross-database transactions are not supported
		var databaseName = qs.Compiler().DatabaseName()
		if dbName != databaseName {
			panicFromNewQuerySet = errors.CrossDatabaseTransaction.WithCause(fmt.Errorf(
				"RunInTransaction: database name mismatch, expected %q, got %q",
				dbName, databaseName,
			))
			panic(panicFromNewQuerySet)
		}

		return qs.WithContext(ctx)
	}

	// rollback the transaction if anything bad happens or the transaction is not committed.
	// this should do nothing if the transaction is already committed.
	defer func() {
		if rec := recover(); rec != nil {
			logger.Errorf("RunInTransaction: panic recovered: %v", rec)
		}

		if transaction != nil && !comitted {
			if err := transaction.Rollback(ctx); err != nil {
				logger.Errorf("RunInTransaction: failed to rollback transaction: %v", err)
			}
		}
	}()

	// if the function returns an error, the transaction will be rolled back
	commit, err := fn(ctx, newQuerySetFunc)
	if err != nil {
		return errors.Wrap(err, "RunInTransaction: function returned an error")
	}

	if commit {
		// commit the transaction if everything went well
		err = transaction.Commit(ctx)
		if err != nil {
			return errors.Wrap(err, "RunInTransaction: failed to commit transaction")
		}
		comitted = true
	}

	return nil
}
