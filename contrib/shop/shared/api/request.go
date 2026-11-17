package api

import (
	"fmt"
	"net/http"
	"sync"
)

type APIRequest interface {
	Started() error
	Finished() error
	HttpRequest() *http.Request
}

type BaseAPIRequest struct {
	*http.Request
}

func (s *BaseAPIRequest) Started() error {
	return nil
}

func (s *BaseAPIRequest) Finished() error {
	return nil
}

func (s *BaseAPIRequest) HttpRequest() *http.Request {
	return s.Request
}

type RequestRegistry struct {
	fallback func(*http.Request, string) APIRequest
	registry map[string]func(*http.Request, string) APIRequest
	mu       *sync.RWMutex
}

func NewRequestRegistry(fallback func(*http.Request, string) APIRequest) RequestRegistry {
	return RequestRegistry{
		fallback: fallback,
		registry: make(map[string]func(*http.Request, string) APIRequest),
		mu:       &sync.RWMutex{},
	}
}

func (reg *RequestRegistry) RequestStarted(r *http.Request, apiName string) (APIRequest, error) {
	reg.mu.RLock()
	newRequest, ok := reg.registry[apiName]
	reg.mu.RUnlock()

	if ok {
		if req := newRequest(r, apiName); req != nil {
			return req, req.Started()
		}
	}

	return &BaseAPIRequest{
		Request: r,
	}, nil
}

func (reg *RequestRegistry) RegisterNewAPIRequest(apiName string, newRequest func(*http.Request, string) APIRequest) {
	reg.mu.Lock()
	defer reg.mu.Unlock()

	_, ok := reg.registry[apiName]
	if ok {
		panic(fmt.Sprintf("request type for %q already registered", apiName))
	}

	reg.registry[apiName] = newRequest
}
