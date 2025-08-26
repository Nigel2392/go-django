package admin

import (
	"context"
	"net/http"

	"github.com/Nigel2392/go-django/src/contrib/admin/components"
	"github.com/Nigel2392/go-django/src/contrib/admin/components/menu"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/goldcrest"
	"github.com/a-h/templ"
	"github.com/justinas/nosurf"
)

var _ ctx.Context = (*adminContext)(nil)
var _ ctx.ContextWithRequest = (*adminContext)(nil)

type BreadCrumb struct {
	Title string
	URL   string
}

type Action struct {
	Icon   string
	Target string
	Title  string
	URL    string
}

type boundSidePanel struct {
	request *http.Request
	icon    func(ctx context.Context) string
	title   func(ctx context.Context) string
	content templ.Component
	media   func() media.Media
}

type PageOptions struct {
	Request     *http.Request
	TitleFn     func(ctx context.Context) string
	SubtitleFn  func(ctx context.Context) string
	MediaFn     func() media.Media
	BreadCrumbs []BreadCrumb
	Actions     []Action
	Buttons     []components.ShowableComponent
	SidePanels  []menu.SidePanel
}

func (p *PageOptions) Title() string {
	if p.TitleFn == nil {
		return ""
	}
	return p.TitleFn(p.Request.Context())
}

func (p *PageOptions) Subtitle() string {
	if p.SubtitleFn == nil {
		return ""
	}
	return p.SubtitleFn(p.Request.Context())
}

func (p *PageOptions) Media() media.Media {
	var media media.Media = media.NewMedia()
	if p.MediaFn != nil {
		media = media.Merge(p.MediaFn())
	}
	for _, panel := range p.GetSidePanels().Panels {
		media = media.Merge(panel.Media())
	}
	return media
}

func (p *PageOptions) GetBreadCrumbs() []BreadCrumb {
	var breadCrumbs = p.BreadCrumbs
	if breadCrumbs == nil {
		breadCrumbs = make([]BreadCrumb, 0)
	}

	var hooks = goldcrest.Get[RegisterBreadCrumbHookFunc](RegisterNavBreadCrumbHook)
	for _, hook := range hooks {
		var crumbs = hook(p.Request, AdminSite)
		breadCrumbs = append(breadCrumbs, crumbs...)
	}

	return breadCrumbs
}

func (p *PageOptions) GetActions() []Action {
	var actions = p.Actions
	if actions == nil {
		actions = make([]Action, 0)
	}

	var hooks = goldcrest.Get[RegisterNavActionHookFunc](RegisterNavActionHook)
	for _, hook := range hooks {
		var acts = hook(p.Request, AdminSite)
		actions = append(actions, acts...)
	}

	return actions
}

func (p *PageOptions) GetSidePanels() *menu.SidePanels {
	var sidePanels = p.SidePanels
	if len(sidePanels) == 0 {
		sidePanels = make([]menu.SidePanel, 0)
	}
	return &menu.SidePanels{
		Panels: sidePanels,
	}
}

type adminContext struct {
	Page    *PageOptions
	Site    *AdminApplication
	request *http.Request
	Context ctx.Context
}

func NewContext(request *http.Request, site *AdminApplication, context ctx.Context) *adminContext {
	if context == nil {
		context = ctx.RequestContext(request)
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
	if v, ok := value.(ctx.Editor); ok {
		v.EditContext(key, c)
		return
	}
	switch key {
	case "site", "Site":
		c.Site = value.(*AdminApplication)
		return
	case "page", "Page":
		c.Page = value.(*PageOptions)
	}

	c.Context.Set(key, value)
}

func (c *adminContext) Data() map[string]interface{} {
	var data = c.Context.Data()
	if c.Page != nil {
		data["page"] = c.Page
	}
	if c.Site != nil {
		data["site"] = c.Site
	}
	return data
}

func (c *adminContext) SetPage(page PageOptions) {
	c.Page = &page
	c.Page.Request = c.request
}

func (c *adminContext) Request() *http.Request {
	return c.request
}

func (c *adminContext) CsrfToken() string {
	return nosurf.Token(c.request)
}
