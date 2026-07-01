package cache

import "context"

type transactionContextKey[T any] struct{}

func Typed_ContextWithTransaction[T any](ctx context.Context, tx TypedTransaction[T]) context.Context {
	return context.WithValue(ctx, transactionContextKey[T]{}, tx)
}

func Typed_TransactionFromContext[T any](ctx context.Context) (t TypedTransaction[T], ok bool) {
	var val = ctx.Value(transactionContextKey[T]{})
	if val == nil {
		return nil, false
	}
	t, ok = val.(TypedTransaction[T])
	return t, ok
}

func ContextWithTransaction(ctx context.Context, tx Transaction) context.Context {
	return Typed_ContextWithTransaction(ctx, tx)
}

func TransactionFromContext(ctx context.Context) (t Transaction, ok bool) {
	return Typed_TransactionFromContext[any](ctx)
}

func transactionOrDefault(ctx context.Context) Cache {
	t, ok := TransactionFromContext(ctx)
	if ok && t.InTransaction() {
		return t
	}
	return Default()
}
