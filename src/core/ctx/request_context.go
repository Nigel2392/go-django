package ctx

import "net/http"

type HTTPRequestContext struct {
	*StructContext
	HttpRequest *http.Request
	CsrfToken   string
}

func RequestContext(r *http.Request) *HTTPRequestContext {
	var request = &HTTPRequestContext{
		HttpRequest: r,
	}

	var c = NewStructContext(request)
	request.StructContext = c.(*StructContext)
	request.StructContext.DeniesContext = true // prevent infinite recursion
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
	c.StructContext.Set(key, value)
}
