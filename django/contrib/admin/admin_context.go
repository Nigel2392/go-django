package admin

import (
	"net/http"

	"github.com/Nigel2392/django/core"
	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/django/core/ctx"
	"github.com/Nigel2392/django/core/tpl"
	"github.com/Nigel2392/django/forms/media"
	"github.com/justinas/nosurf"
)

var _ ctx.Context = (*adminContext)(nil)
var _ tpl.RequestContext = (*adminContext)(nil)

type BreadCrumb struct {
	Title string
	URL   string
}

type PageOptions struct {
	TitleFn     func() string
	SubtitleFn  func() string
	MediaFn     func() media.Media
	BreadCrumbs []BreadCrumb
}

func (p *PageOptions) Title() string {
	if p.TitleFn == nil {
		return ""
	}
	return p.TitleFn()
}

func (p *PageOptions) Subtitle() string {
	if p.SubtitleFn == nil {
		return ""
	}
	return p.SubtitleFn()
}

type adminContext struct {
	Page    *PageOptions
	Site    *AdminApplication
	request *http.Request
	Context ctx.Context
}

func NewContext(request *http.Request, site *AdminApplication, context ctx.Context) *adminContext {
	if context == nil {
		context = core.Context(request)
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
	case "page", "Page":
		return c.Page
	}

	return c.Context.Get(key)
}

func (c *adminContext) Set(key string, value interface{}) {
	switch key {
	case "site", "Site":
		c.Site = value.(*AdminApplication)
		return
	case "page", "Page":
		c.Page = value.(*PageOptions)
	}
	c.Context.Set(key, value)
}

func (c *adminContext) SetPage(page PageOptions) {
	c.Page = &page
}

func (c *adminContext) Request() *http.Request {
	return c.request
}

func (c *adminContext) CsrfToken() string {
	return nosurf.Token(c.request)
}
