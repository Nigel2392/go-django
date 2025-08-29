package admin

import (
	"net/http"

	"github.com/Nigel2392/go-django/src/contrib/admin/components/menu"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/trans"
)

func SidePanelFilters(request *http.Request, filters any, pageObject any, hidden ...func(p *menu.BaseSidePanel, r *http.Request) bool) menu.SidePanel {
	return &menu.BaseSidePanel{
		ID:           "filters",
		Ordering:     100,
		Request:      request,
		TemplateName: "admin/shared/side_panels/filter_panel.tmpl",
		PanelLabel:   trans.S("Filters"),
		Hidden: func(p *menu.BaseSidePanel, r *http.Request) bool {
			for _, fn := range hidden {
				if fn(p, r) {
					return true
				}
			}
			return false
		},
		Context: func(p *menu.BaseSidePanel, r *http.Request, c ctx.Context) ctx.Context {
			c.Set("filter", filters)
			c.Set("view_paginator_object", pageObject)
			return c
		},
	}
}
