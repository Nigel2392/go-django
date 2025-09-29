package admin

import (
	"context"
	"html/template"
	"net/http"
	"strings"

	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
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

func NewPanelBoundForm(ctx context.Context, request *http.Request, instance attrs.Definer, form forms.Form, boundform forms.BoundForm, panels []Panel, formsets FormSetObject) *PanelBoundForm {
	var (
		boundForm = &PanelBoundForm{
			BoundForm:   boundform,
			Panels:      panels,
			BoundPanels: make([]BoundPanel, 0),
			Context:     ctx,
		}
		boundFields = boundform.Fields()
		boundMap    = make(map[string]forms.BoundField)
	)

	for _, field := range boundFields {
		boundMap[field.Input().Name()] = field
	}

	if len(panels) > 0 {
		for _, panel := range BindPanels(panels, request, make(map[string]int), form, boundForm.Context, instance, boundMap, formsets) {
			boundForm.BoundPanels = append(
				boundForm.BoundPanels, panel,
			)
		}
	} else {
		var fields []fields.Field
		if len(form.FieldOrder()) > 0 {
			for _, name := range form.FieldOrder() {
				var f, _ = form.Field(name)
				fields = append(fields, f)
			}
		} else {
			fields = form.Fields()
		}

		var idx int
		var m, ok = formsets.(FormSetMap)
		if !ok {
			m = nil
		}

		for i, field := range fields {
			var panel = FieldPanel(field.Name())
			var boundPanel = panel.Bind(request, make(map[string]int), form, boundForm.Context, instance, boundMap, m[panelPathPart(panel, i)])
			if boundPanel == nil {
				continue
			}

			boundForm.BoundPanels = append(
				boundForm.BoundPanels, boundPanel,
			)

			idx++
		}
	}

	return boundForm
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
