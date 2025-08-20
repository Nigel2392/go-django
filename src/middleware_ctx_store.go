package django

import (
	"context"
	"net/http"

	"github.com/Nigel2392/mux"
)

type middlewareContextStoreKey struct{}

type ContextDataStore interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{})
	Delete(key string) bool
}

type datastore struct {
	data map[string]interface{}
}

func (d *datastore) Get(key string) (interface{}, bool) {
	var val, ok = d.data[key]
	return val, ok
}

func (d *datastore) Set(key string, value interface{}) {
	d.data[key] = value
}

func (d *datastore) Delete(key string) bool {
	if _, ok := d.data[key]; !ok {
		return false
	}
	delete(d.data, key)
	return true
}

func newDatastore() ContextDataStore {
	return &datastore{
		data: make(map[string]interface{}),
	}
}

func GetContextDataStore(requestOrContext any) ContextDataStore {
	var ctx context.Context
	switch v := requestOrContext.(type) {
	case *http.Request:
		ctx = v.Context()
	case context.Context:
		ctx = v
	default:
		return nil
	}
	return ctx.Value(middlewareContextStoreKey{}).(ContextDataStore)
}

// ContextDataStoreMiddleware is a middleware that provides a ContextDataStore for each request.
//
// This is a global datastore, on a per request basis - the underlying datastructure is a map[string]interface{}.
//
// This allows for some flexibility with caching from lower context levels.
func ContextDataStoreMiddleware(next mux.Handler) mux.Handler {
	return mux.NewHandler(func(w http.ResponseWriter, r *http.Request) {
		var store ContextDataStore = newDatastore()
		var ctx = context.WithValue(r.Context(), middlewareContextStoreKey{}, store)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
