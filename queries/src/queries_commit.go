package queries

import "context"

type queryContextKey struct {
	name string
}

var commitContextKey = queryContextKey{"queries.db.disallow_commit"}

func IsCommitContext(ctx context.Context) bool {
	v, ok := ctx.Value(commitContextKey).(bool)
	return !ok || !v
}

func CommitContext(ctx context.Context, canCommit bool) context.Context {
	return context.WithValue(ctx, commitContextKey, !canCommit)
}
