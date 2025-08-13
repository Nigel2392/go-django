package admin

import (
	"context"
	"html/template"
	"net/http"
	"strings"

	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/elliotchance/orderedmap/v2"
)

type PanelBoundForm struct {
	forms.BoundForm
	BoundPanels []BoundPanel
	Panels      []Panel
	Context     context.Context
	Request     *http.Request
}

func (b *PanelBoundForm) AsP() template.HTML {
	var html = new(strings.Builder)
	for _, panel := range b.BoundPanels {
		var component = panel.Component()
		component.Render(b.Context, html)
	}
	return template.HTML(html.String())
}
func (b *PanelBoundForm) AsUL() template.HTML {
	var html = new(strings.Builder)
	for _, panel := range b.BoundPanels {
		var component = panel.Component()
		component.Render(b.Context, html)
	}
	return template.HTML(html.String())
}
func (b *PanelBoundForm) Media() media.Media {
	return b.BoundForm.Media()
}
func (b *PanelBoundForm) Fields() []forms.BoundField {
	return b.BoundForm.Fields()
}
func (b *PanelBoundForm) ErrorList() []error {
	return b.BoundForm.ErrorList()
}
func (b *PanelBoundForm) Errors() *orderedmap.OrderedMap[string, []error] {
	return b.BoundForm.Errors()
}
