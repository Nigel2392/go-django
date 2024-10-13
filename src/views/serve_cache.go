package views

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"

	"github.com/Nigel2392/go-django/src/core/cache"
	"github.com/Nigel2392/go-django/src/core/logger"
)

func fnvhash(s string) string {
	var h = fnv.New128a()
	h.Write([]byte(s))
	var sum = h.Sum(nil)
	return fmt.Sprintf("%x", sum)
}

type byteArray []byte

func (b byteArray) MarshalJSON() ([]byte, error) {
	return []byte(base64.StdEncoding.EncodeToString(b)), nil
}

func (b *byteArray) UnmarshalJSON(data []byte) error {
	var decoded, err = base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return err
	}
	*b = decoded
	return nil
}

type CachedResponse struct {
	ResponseStatusCode int         `json:"status_code"`
	ResponseHeader     http.Header `json:"header"`
	ResponseBody       byteArray   `json:"body"`
}

func newCachedResponse() *CachedResponse {
	return &CachedResponse{
		ResponseHeader: make(http.Header),
	}
}

type responseWriter struct {
	Status  int
	Headers http.Header
	Body    *bytes.Buffer
}

func (w *responseWriter) Header() http.Header {
	return w.Headers
}

func (w *responseWriter) Write(data []byte) (int, error) {
	return w.Body.Write(data)
}

func (w *responseWriter) WriteHeader(status int) {
	w.Status = status
}

func buildRequestCacheKey(req *http.Request) string {
	var hash = fnv.New128a()
	for key, values := range req.Header {
		hash.Write([]byte(key))
		for _, value := range values {
			if value == "" {
				continue
			}
			hash.Write([]byte(value))
		}
	}
	url := req.URL.String()
	var sum = hash.Sum(nil)
	return fmt.Sprintf("views.buildRequestCacheKey.%s.%x.%x", req.Method, sum, fnvhash(url))
}

// Cache caches the response of a view for a given duration.
func Cache(view http.Handler, duration cache.Duration, cacheBackends ...string) http.Handler {
	if duration == 0 {
		duration = cache.Infinity
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var cacheKey = buildRequestCacheKey(req)
		var cacheBackend = cache.GetCache(cacheBackends...)

		if !cacheBackend.Has(cacheKey) {
			logger.Debugf(
				"Key does not exist in cache: %s", cacheKey,
			)
			var buf = new(bytes.Buffer)
			var cachedResponse = newCachedResponse()
			var responseWriter = &responseWriter{
				Headers: make(http.Header),
				Body:    buf,
			}

			view.ServeHTTP(responseWriter, req)

			cachedResponse.ResponseStatusCode = responseWriter.Status
			cachedResponse.ResponseHeader = responseWriter.Headers
			cachedResponse.ResponseBody = buf.Bytes()

			logger.Debugf(
				"Caching response: %s", cacheKey,
			)
			var err = cacheBackend.Set(cacheKey, cachedResponse, duration)
			if err != nil {
				logger.Errorf(
					"Error caching response: %s", err,
				)
			}

			w.WriteHeader(responseWriter.Status)

			for key, values := range responseWriter.Headers {
				for _, value := range values {
					w.Header().Add(key, value)
				}
			}

			_, err = io.Copy(w, buf)
			if err != nil {
				logger.Errorf(
					"Error copying response: %s", err,
				)
			}
			return
		}

		logger.Debugf(
			"Retrieving cached response: %s", cacheKey,
		)
		var response, err = cacheBackend.Get(cacheKey)
		if err != nil {
			logger.Errorf(
				"Error getting cached response: %s", err,
			)
			view.ServeHTTP(w, req)
			return
		}

		var cached = response.(*CachedResponse)
		w.WriteHeader(cached.ResponseStatusCode)

		for key, values := range cached.ResponseHeader {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}

		_, err = w.Write(cached.ResponseBody)
		if err != nil {
			logger.Errorf(
				"Error writing cached response: %s", err,
			)
		}
	})
}
