package django

import (
	"context"
	"net/http"

	"github.com/Nigel2392/mux"
)

type middlewareContextStoreKey struct{}

type datastore struct {
	data map[any]interface{}
}

func (d *datastore) Get(key any) (interface{}, bool) {
	if d == nil {
		return nil, false
	}
	var val, ok = d.data[key]
	return val, ok
}

func (d *datastore) Set(key any, value interface{}) {
	if d == nil {
		return
	}
	d.data[key] = value
}

func (d *datastore) Delete(key any) bool {
	if d == nil {
		return false
	}
	if _, ok := d.data[key]; !ok {
		return false
	}
	delete(d.data, key)
	return true
}

func newDatastore() *datastore {
	return &datastore{
		data: make(map[any]interface{}),
	}
}

func ContextDataStoreGet(requestOrContext any, key any, defaultValue ...interface{}) interface{} {
	var ctx context.Context
	switch v := requestOrContext.(type) {
	case *http.Request:
		ctx = v.Context()
	case context.Context:
		ctx = v
	default:
		panic("ContextDataStoreGet: invalid type, must be *http.Request or context.Context")
	}

	var def any
	if len(defaultValue) > 0 {
		def = defaultValue[0]
	} else if len(defaultValue) > 1 {
		panic("ContextDataStoreGet: too many default values provided")
	}
	var val, ok = ctx.Value(middlewareContextStoreKey{}).(*datastore).Get(key)
	if !ok {
		return def
	}
	return val
}

func ContextDataStoreGetOK(requestOrContext any, key any) (interface{}, bool) {
	var ctx context.Context
	switch v := requestOrContext.(type) {
	case *http.Request:
		ctx = v.Context()
	case context.Context:
		ctx = v
	default:
		panic("ContextDataStoreGet: invalid type, must be *http.Request or context.Context")
	}
	return ctx.Value(middlewareContextStoreKey{}).(*datastore).Get(key)
}

func ContextDataStoreSet(requestOrContext any, key any, value interface{}) {
	var ctx context.Context
	switch v := requestOrContext.(type) {
	case *http.Request:
		ctx = v.Context()
	case context.Context:
		ctx = v
	default:
		panic("ContextDataStoreSet: invalid type, must be *http.Request or context.Context")
	}
	ctx.Value(middlewareContextStoreKey{}).(*datastore).Set(key, value)
}

func ContextDataStoreDelete(requestOrContext any, key string) bool {
	var ctx context.Context
	switch v := requestOrContext.(type) {
	case *http.Request:
		ctx = v.Context()
	case context.Context:
		ctx = v
	default:
		panic("ContextDataStoreDelete: invalid type, must be *http.Request or context.Context")
	}
	return ctx.Value(middlewareContextStoreKey{}).(*datastore).Delete(key)
}

func ContextDataStore(requestOrContext any) map[any]interface{} {
	var ctx context.Context
	switch v := requestOrContext.(type) {
	case *http.Request:
		ctx = v.Context()
	case context.Context:
		ctx = v
	default:
		panic("ContextDataStore: invalid type, must be *http.Request or context.Context")
	}
	return ctx.Value(middlewareContextStoreKey{}).(*datastore).data
}

// ContextDataStoreMiddleware is a middleware that provides a ContextDataStore for each request.
//
// This is a global datastore, on a per request basis - the underlying datastructure is a map[string]interface{}.
//
// This allows for some flexibility with caching from lower context levels.
func ContextDataStoreMiddleware(next mux.Handler) mux.Handler {
	return mux.NewHandler(func(w http.ResponseWriter, r *http.Request) {
		var ctx = context.WithValue(r.Context(), middlewareContextStoreKey{}, newDatastore())
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
