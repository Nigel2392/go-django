package drivers

import (
	"context"
	"time"
)

var (
	queryContextKey = dbContextKey{"db.query.context"}
)

func ContextWithQueryInfo(ctx context.Context) (context.Context, *QueryInformation) {
	var qi = &QueryInformation{
		Queries: make([]*Query, 0),
		Start:   time.Now(),
	}
	return context.WithValue(ctx, queryContextKey, qi), qi
}

func ContextQueryInfo(ctx context.Context) (*QueryInformation, bool) {
	var v = ctx.Value(queryContextKey)
	if v == nil {
		return nil, false
	}

	q, ok := v.(*QueryInformation)
	return q, ok
}

func ContextQueryExec[T any](ctx context.Context, driver string, query string, args []any, flags QueryFlag, fn func(ctx context.Context, query string, args ...any) (T, error)) (T, error) {
	var qi, ok = ContextQueryInfo(ctx)
	if !ok {
		return fn(ctx, query, args...)
	}

	var start = time.Now()
	var result, err = fn(ctx, query, args...)
	qi.Queries = append(qi.Queries, &Query{
		Context:   qi,
		Driver:    driver,
		Query:     query,
		Args:      args,
		Error:     err,
		Start:     start,
		TimeTaken: time.Since(start),
		Flags:     flags,
	})
	return result, err
}
