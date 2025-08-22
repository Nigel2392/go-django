package images

import (
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/forms/media"
)

var _ admin.StringRenderer = &SearchComponent{}

type SearchComponent struct {
	View    *admin.BoundSearchView
	Objects []attrs.Definer
}

func (c *SearchComponent) Media() media.Media {
	var m = media.NewMedia()
	m.AddCSS(media.CSS(django.Static("images/css/admin.css")))
	return m
}

func (c *SearchComponent) Render() string {
	var context = ctx.RequestContext(c.View.R)
	context.Set("instances", c.Objects)
	var html, err = tpl.Render(context, "images/image_search_list.tmpl")
	if err != nil {
		logger.Errorf("Failed to render image search list template: %v", err)
		assert.Fail("Failed to render image search list template: %v", err)
	}
	return string(html)
}
