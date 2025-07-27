package ctx

import (
	"net/http"

	"github.com/justinas/nosurf"
)

type HTTPRequestContext struct {
	Context
	HttpRequest *http.Request
	CsrfToken   string
}

func RequestContext(r *http.Request) *HTTPRequestContext {
	var request = &HTTPRequestContext{
		HttpRequest: r,
		Context:     NewContext(nil),
		CsrfToken:   nosurf.Token(r),
	}

	return request
}

func (c *HTTPRequestContext) Request() *http.Request {
	return c.HttpRequest
}

func (c *HTTPRequestContext) CSRFToken() string {
	return c.CsrfToken
}

func (c *HTTPRequestContext) Set(key string, value any) {
	if v, ok := value.(Editor); ok {
		v.EditContext(key, c)
		return
	}

	c.Context.Set(key, value)
}

func (c *HTTPRequestContext) Get(key string) any {
	switch key {
	case "csrf_token", "CsrfToken", "CSRFToken":
		return c.CsrfToken
	case "request", "Request":
		return c.HttpRequest
	}
	return c.Context.Get(key)
}
