package embedder

import (
	"context"
	"unsafe"
)

type rootValueContextKey struct{}

var (
	contextKey    any = rootValueContextKey{}
	contextKeyTyp     = (*eface)(unsafe.Pointer(&contextKey)).typ
)

func RootContext(ctx context.Context) *RootValueContext {
	rf := ctx.Value(contextKey)
	if rf == nil {
		return nil
	}
	return rf.(*RootValueContext)
}

type RootValueContext struct {
	context.Context
	Root any
}

func (r *RootValueContext) Value(key any) any {
	// faster than key == contextKey
	if (*eface)(unsafe.Pointer(&key)).typ == contextKeyTyp {
		return r
	}

	return r.Context.Value(key)
}
