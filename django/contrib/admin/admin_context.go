package admin

import (
	"net/http"

	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/django/core/ctx"
	"github.com/Nigel2392/django/core/tpl"
)

var _ ctx.Context = (*adminContext)(nil)
var _ tpl.RequestContext = (*adminContext)(nil)

type PageOptions struct {
	Title string
}

type adminContext struct {
	ctx.Context
	Page    PageOptions
	Site    *AdminApplication
	request *http.Request
}

func NewContext(request *http.Request, site *AdminApplication, context ctx.Context) *adminContext {
	if context == nil {
		context = ctx.NewContext(nil)
	}

	assert.False(
		site == nil,
		"Site must be provided to AdminContext",
	)

	var c = &adminContext{
		Context: context,
		Site:    site,
		request: request,
	}

	return c
}

func (c *adminContext) Get(key string) interface{} {
	switch key {
	case "site", "Site":
		return c.Site
	}

	return c.Context.Get(key)
}

func (c *adminContext) Request() *http.Request {
	return c.request
}
