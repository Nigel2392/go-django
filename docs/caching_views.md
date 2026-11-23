# Caching Views

The `github.com/Nigel2392/go-django/src/views` package includes a `Cache` wrapper (`src/views/serve_cache.go`) that caches HTTP responses in a [`cache`](./cache.md) backend for a given `time.Duration`.

---

## Experimental Status and Known Issues

`views.Cache` is currently marked `TODO: FIX REQUEST CACHE IMPLEMENTATION` in `src/views/serve_cache.go` and has several known runtime caveats:

1. **`WriteHeader` is called before copying response headers**: On both cache misses and cache hits, `w.WriteHeader(...)` is called *before* copying headers into `w.Header()`. Under Go's `net/http.ResponseWriter`, header mutations after `WriteHeader` are ignored, so cached `Content-Type` and other response headers are not sent to the client.
2. **Status `0` when the inner handler omits `WriteHeader`**: If the wrapped handler calls `w.Write(...)` without explicitly calling `w.WriteHeader(code)`, `responseWriter.Status` remains `0` and `w.WriteHeader(0)` is called on the underlying `http.ResponseWriter`.
3. **Nondeterministic cache key hashing**: `buildRequestCacheKey` iterates directly over `req.Header` (a Go `map`), writing keys and non-empty values into an FNV-128a hash without sorting keys first. Map iteration order in Go is randomized, so identical requests with multiple headers can hash to different cache keys.
4. **Invalid `byteArray.MarshalJSON`**: `CachedResponse.ResponseBody` encodes raw base64 bytes without surrounding JSON quotes, so marshaling a `CachedResponse` to JSON (for distributed backends such as Redis) produces invalid JSON.

Because of these issues, `views.Cache` should currently be treated as experimental and avoided in production until fixed.

---

## API Reference

### `CachedResponse`

```go
type CachedResponse struct {
	ResponseStatusCode int         `json:"status_code"`
	ResponseHeader     http.Header `json:"header"`
	ResponseBody       byteArray   `json:"body"`
}
```

### `Cache`

```go
func Cache(view http.Handler, duration time.Duration, cacheBackends ...string) http.Handler
```

- **`view http.Handler`**: The HTTP handler whose response will be cached.
- **`duration time.Duration`**: Time-to-live (TTL) for the cached entry. If `0` is passed, `views.Cache` substitutes `cache.Infinity`.
- **`cacheBackends ...string`**: Optional backend names passed to `cache.GetCache(cacheBackends...)`. When omitted, the default cache backend (`"default"`) is used.

---

## Example Usage

Ensure a cache backend is registered (either via `cache.SetDefault(...)` or `cache.RegisterCache(name, backend)`) before serving requests:

```go
package main

import (
	"net/http"
	"time"

	"github.com/Nigel2392/cache"
	"github.com/Nigel2392/go-django/src/views"
)

func myView(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hello, World!"))
}

func main() {
	cache.SetDefault(cache.NewMemoryCache(time.Minute))

	http.Handle("/cached-view", views.Cache(http.HandlerFunc(myView), 10*time.Minute))
	http.ListenAndServe(":8080", nil)
}
```
