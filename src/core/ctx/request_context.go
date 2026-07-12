package ctx

import (
	"net/http"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
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

func (c *HTTPRequestContext) Clone(values ...interface{}) (Context, error) {
	var subCopy, _ = c.Context.Clone()
	var copy = &HTTPRequestContext{
		Context:     subCopy,
		HttpRequest: c.HttpRequest.Clone(c.HttpRequest.Context()),
		CsrfToken:   c.CsrfToken,
	}

	var other, err = TemplateDictFunc(values...)
	if err != nil {
		return nil, errors.Wrap(err, "HTTPRequestContext")
	}

	for k, v := range other {
		copy.Set(k, v)
	}

	return copy, nil
}
