package views_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/Nigel2392/go-django/src/core/cache"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/views"

	_ "unsafe"
)

//go:linkname buildRequestCacheKey github.com/Nigel2392/go-django/src/views.buildRequestCacheKey
func buildRequestCacheKey(r *http.Request) string

//go:linkname newCachedResponse github.com/Nigel2392/go-django/src/views.newCachedResponse
func newCachedResponse() *views.CachedResponse

// Mock handler to return a fixed response.
func mockHandler(statusCode int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		w.Write([]byte(body))
	}
}

func init() {
	// Register the cache backend
	cache.RegisterCache("memory", cache.NewMemoryCache(5*time.Second))

	logger.Setup(&logger.Logger{
		Level:      logger.DBG,
		OutputTime: true,
	})

	logger.SetOutput(logger.OutputAll, os.Stdout)

	fmt.Println("Initialized logger")
}

// TestCacheMiss tests when the response is not cached (cache miss).
func TestCacheMiss(t *testing.T) {
	// Use NewMemoryCache for the cache backend.
	cacheBackend := cache.GetCache("memory")
	handler := views.Cache(mockHandler(http.StatusOK, "Hello, world!"), time.Millisecond*30, "memory")

	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	// Assertions
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	if string(body) != "Hello, world!" {
		t.Errorf("Expected body to be %s, got %s", "Hello, world!", body)
	}

	// Check if the response is cached
	cacheKey := buildRequestCacheKey(req)
	if !cacheBackend.Has(cacheKey) {
		t.Error("Expected response to be cached")
	}

	time.Sleep(time.Millisecond * 60)
}

// TestCacheHit tests when the response is cached (cache hit).
func TestCacheHit(t *testing.T) {
	cacheBackend := cache.GetCache("memory")
	cachedResp := &views.CachedResponse{
		ResponseStatusCode: http.StatusOK,
		ResponseHeader:     http.Header{"Content-Type": []string{"text/plain"}},
		ResponseBody:       []byte("Cached response"),
	}
	cacheKey := buildRequestCacheKey(httptest.NewRequest("GET", "http://example.com/foo", nil))
	cacheBackend.Set(cacheKey, cachedResp, time.Millisecond*50)

	handler := views.Cache(mockHandler(http.StatusOK, "Hello, world!"), time.Millisecond*50, "memory")

	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	// Assertions
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	if string(body) != "Cached response" {
		t.Errorf("Expected body to be %s, got %s", "Cached response", body)
	}

	time.Sleep(time.Millisecond * 60)
}

// TestCacheDifferentMethods ensures that different HTTP methods are cached separately.
func TestCacheDifferentMethods(t *testing.T) {
	cacheBackend := cache.GetCache("memory")

	handler := views.Cache(mockHandler(http.StatusOK, "GET response"), time.Millisecond*50, "memory")
	reqGet := httptest.NewRequest("GET", "http://example.com/foo", nil)
	wGet := httptest.NewRecorder()
	handler.ServeHTTP(wGet, reqGet)

	cacheKeyGet := buildRequestCacheKey(reqGet)
	if !cacheBackend.Has(cacheKeyGet) {
		t.Error("Expected GET request to be cached")
	}

	handler = views.Cache(mockHandler(http.StatusOK, "POST response"), time.Millisecond*50, "memory")
	reqPost := httptest.NewRequest("POST", "http://example.com/foo", nil)
	wPost := httptest.NewRecorder()
	handler.ServeHTTP(wPost, reqPost)

	cacheKeyPost := buildRequestCacheKey(reqPost)
	if !cacheBackend.Has(cacheKeyPost) {
		t.Error("Expected POST request to be cached")
	}

	// Check if GET and POST cache keys are different
	if cacheKeyGet == cacheKeyPost {
		t.Error("Expected different cache keys for GET and POST")
	}

	time.Sleep(time.Millisecond * 60)
}

// TestCacheExpiration tests that the cache expires after the given duration.
func TestCacheExpiration(t *testing.T) {
	cacheBackend := cache.GetCache("memory")

	handler := views.Cache(mockHandler(http.StatusOK, "Response"), time.Millisecond*50, "memory")

	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	cacheKey := buildRequestCacheKey(req)
	if !cacheBackend.Has(cacheKey) {
		t.Error("Expected response to be cached")
	}

	// Wait for cache to expire
	time.Sleep(time.Millisecond * 60)

	if cacheBackend.Has(cacheKey) {
		t.Error("Expected cache to expire")
	}
}
