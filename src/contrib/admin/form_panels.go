package admin

import (
	"context"
	"fmt"
	"html/template"
	"net/http"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
)

var (
	_ forms.Form                          = (*AdminForm[modelforms.ModelForm[attrs.Definer]])(nil)
	_ modelforms.ModelForm[attrs.Definer] = (*AdminModelForm[modelforms.ModelForm[attrs.Definer], attrs.Definer])(nil)
)

type PanelContext struct {
	*ctx.HTTPRequestContext
	Panel      Panel
	BoundPanel BoundPanel
}

func NewPanelContext(r *http.Request, panel Panel, boundPanel BoundPanel) *PanelContext {
	return &PanelContext{
		HTTPRequestContext: ctx.RequestContext(r),
		Panel:              panel,
		BoundPanel:         boundPanel,
	}
}

type Panel interface {
	Fields() []string
	ClassName() string
	Class(classes string) Panel
	Bind(r *http.Request, form forms.Form, ctx context.Context, boundFields map[string]forms.BoundField) BoundPanel
}

func PanelClass(className string, panel Panel) Panel {
	return panel.Class(className)
}

type fieldPanel struct {
	fieldname string
	classname string
}

func (f *fieldPanel) Fields() []string {
	return []string{f.fieldname}
}

func (f *fieldPanel) ClassName() string {
	return f.classname
}

func (f *fieldPanel) Class(classname string) Panel {
	f.classname = classname
	return f
}

func (f *fieldPanel) Bind(r *http.Request, form forms.Form, ctx context.Context, boundFields map[string]forms.BoundField) BoundPanel {
	var bf, ok = boundFields[f.fieldname]
	if !ok {
		panic(fmt.Sprintf("Field %s not found in bound fields: %v", f.fieldname, boundFields))
	}

	return &BoundFormPanel[forms.Form, *fieldPanel]{
		Panel:      f,
		Form:       form,
		Context:    ctx,
		BoundField: bf,
		Request:    r,
	}
}

func FieldPanel(fieldname string, className ...string) Panel {
	var c string
	if len(className) > 0 {
		c = className[0]
	}
	return &fieldPanel{
		fieldname: fieldname,
		classname: c,
	}
}

type titlePanel struct {
	Panel
	classname string
}

func (t *titlePanel) Class(classname string) Panel {
	t.classname = classname
	return t
}

func (t *titlePanel) ClassName() string {
	return t.classname
}

func (t *titlePanel) Fields() []string {
	return t.Panel.Fields()
}

func (t *titlePanel) Bind(r *http.Request, form forms.Form, ctx context.Context, boundFields map[string]forms.BoundField) BoundPanel {
	return &BoundTitlePanel[forms.Form, *titlePanel]{
		Panel:      t,
		BoundPanel: t.Panel.Bind(r, form, ctx, boundFields),
		Context:    ctx,
		Request:    r,
	}
}

func TitlePanel(panel Panel, classname ...string) Panel {
	var c string
	if len(classname) > 0 {
		c = classname[0]
	}
	return &titlePanel{
		Panel:     panel,
		classname: c,
	}
}

type rowPanel struct {
	panels    []Panel
	Label     func() string
	classname string
}

func (m *rowPanel) Class(classname string) Panel {
	m.classname = classname
	return m
}

func (m *rowPanel) ClassName() string {
	return m.classname
}

func (m *rowPanel) Fields() []string {
	var fields = make([]string, 0)
	for _, panel := range m.panels {
		fields = append(fields, panel.Fields()...)
	}
	return fields
}

func (m *rowPanel) Bind(r *http.Request, form forms.Form, ctx context.Context, boundFields map[string]forms.BoundField) BoundPanel {
	var panels = make([]BoundPanel, 0)
	for _, panel := range m.panels {
		panels = append(panels, panel.Bind(r, form, ctx, boundFields))
	}
	return &BoundRowPanel[forms.Form]{
		LabelFn: m.Label,
		Panel:   m,
		Panels:  panels,
		Context: ctx,
		Request: r,
		Form:    form,
	}
}

func RowPanel(panels ...Panel) Panel {
	return &rowPanel{
		panels: panels,
	}
}

type panelGroup struct {
	panels    []Panel
	classname string
}

func (g *panelGroup) Fields() []string {
	var fields = make([]string, 0, len(g.panels))
	for _, panel := range g.panels {
		fields = append(fields, panel.Fields()...)
	}
	return fields
}

func (g *panelGroup) ClassName() string {
	return g.classname
}

func (g *panelGroup) Class(classname string) Panel {
	g.classname = classname
	return g
}

func (g *panelGroup) Bind(r *http.Request, form forms.Form, ctx context.Context, boundFields map[string]forms.BoundField) BoundPanel {
	var panels = make([]BoundPanel, 0, len(g.panels))
	for _, panel := range g.panels {
		panels = append(panels, panel.Bind(r, form, ctx, boundFields))
	}
	return &BoundPanelGroup[forms.Form]{
		Panel:   g,
		Panels:  panels,
		Context: ctx,
		Request: r,
		Form:    form,
	}
}

func PanelGroup(panels ...Panel) Panel {
	return &panelGroup{
		panels: panels,
	}
}

type AlertType string

const (
	ALERT_INFO    AlertType = "info"
	ALERT_SUCCESS AlertType = "success"
	ALERT_WARNING AlertType = "warning"
	ALERT_ERROR   AlertType = "danger"
)

type AlertPanel struct {
	Type         AlertType
	Label        any
	HTML         any
	TemplateFile string
	Classnames   string
}

func (a *AlertPanel) Fields() []string {
	return []string{}
}

func (a *AlertPanel) ClassName() string {
	return a.Classnames
}

func (a *AlertPanel) Class(classes string) Panel {
	a.Classnames = classes
	return a
}

func (a *AlertPanel) Bind(r *http.Request, form forms.Form, ctx context.Context, boundFields map[string]forms.BoundField) BoundPanel {
	return &BoundAlertPanel[forms.Form]{
		Panel:   a,
		Form:    form,
		Context: ctx,
		Request: r,
	}
}

func (a *AlertPanel) hasLabel() bool {
	return a.Label != nil
}

func (a *AlertPanel) GetType() AlertType {
	if a.Type == "" {
		return ALERT_INFO
	}
	return a.Type
}

func (a *AlertPanel) GetLabel(ctx context.Context) string {
	switch v := a.Label.(type) {
	case string:
		return v
	case func() string:
		return v()
	case func(context.Context) string:
		return v(ctx)
	}

	assert.Fail(
		"AlertPanel.GetLabel: unexpected label type %T", a.Label,
	)
	return ""
}

func (a *AlertPanel) GetHTML(ctx context.Context) template.HTML {
	switch v := a.HTML.(type) {
	case string:
		return template.HTML(v)
	case func() string:
		return template.HTML(v())
	case func(context.Context) string:
		return template.HTML(v(ctx))
	}

	logger.Warnf("AlertPanel.GetHTML: unexpected HTML type %T", a.HTML)
	assert.Fail(
		"AlertPanel.GetHTML: unexpected HTML type %T", a.HTML,
	)
	return ""
}
