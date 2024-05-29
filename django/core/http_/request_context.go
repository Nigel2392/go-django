package http_

import (
	"net/http"

	"github.com/Nigel2392/django/core/ctx"
)

type RequestContext struct {
	*ctx.StructContext
	HttpRequest *http.Request
	CsrfToken   string
}

func Context(r *http.Request) ctx.Context {
	var request = &RequestContext{
		HttpRequest: r,
	}

	var c = ctx.NewStructContext(request)
	request.StructContext = c.(*ctx.StructContext)
	request.StructContext.DeniesContext = true
	return request
}

func (c *RequestContext) Request() *http.Request {
	return c.HttpRequest
}

func (c *RequestContext) CSRFToken() string {
	return c.CsrfToken
}
