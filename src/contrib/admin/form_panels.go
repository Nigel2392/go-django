package admin

import (
	"context"
	"fmt"
	"html/template"
	"net/http"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/a-h/templ"
)

type Panel interface {
	Fields() []string
	ClassName() string
	Class(classes string) Panel
	Bind(r *http.Request, panelIdx []int, form forms.Form, ctx context.Context, boundFields map[string]forms.BoundField) BoundPanel
}

type ValidatorPanel interface {
	Panel
	Validate(r *http.Request, ctx context.Context, form forms.Form, data map[string]any) []error
}

type BoundPanel interface {
	Hidden() bool
	Component() templ.Component
	Render() template.HTML
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

func (f *fieldPanel) Validate(r *http.Request, ctx context.Context, form forms.Form, data map[string]any) []error {
	return nil
}

func (f *fieldPanel) Bind(r *http.Request, panelIdx []int, form forms.Form, ctx context.Context, boundFields map[string]forms.BoundField) BoundPanel {
	var bf, ok = boundFields[f.fieldname]
	if !ok {
		panic(fmt.Sprintf("Field %s not found in bound fields: %v", f.fieldname, boundFields))
	}

	return &BoundFormPanel[forms.Form, *fieldPanel]{
		Panel:      f,
		PanelIndex: panelIdx,
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

func (f *titlePanel) Validate(r *http.Request, ctx context.Context, form forms.Form, data map[string]any) []error {
	if v, ok := f.Panel.(ValidatorPanel); ok {
		return v.Validate(r, ctx, form, data)
	}
	return nil
}

func (t *titlePanel) Bind(r *http.Request, panelIdx []int, form forms.Form, ctx context.Context, boundFields map[string]forms.BoundField) BoundPanel {
	var panel = t.Panel.Bind(r, panelIdx, form, ctx, boundFields)
	if panel == nil {
		return nil
	}

	return &BoundTitlePanel[forms.Form, *titlePanel]{
		Panel:      t,
		PanelIndex: panelIdx,
		BoundPanel: panel,
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
	Label     func(context.Context) string
	HelpText  func(context.Context) string
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

func (f *rowPanel) Validate(r *http.Request, ctx context.Context, form forms.Form, data map[string]any) []error {
	var errs []error
	for _, panel := range f.panels {
		if v, ok := panel.(ValidatorPanel); ok {
			errs = append(errs, v.Validate(r, ctx, form, data)...)
		}
	}
	return errs
}

func (m *rowPanel) Bind(r *http.Request, panelIdx []int, form forms.Form, ctx context.Context, boundFields map[string]forms.BoundField) BoundPanel {
	var panels = make([]BoundPanel, 0, len(m.panels))
	for _, panel := range BindPanels(m.panels, r, panelIdx, form, ctx, boundFields) {
		panels = append(panels, panel)
	}
	return &BoundRowPanel[forms.Form]{
		LabelFn:    m.Label,
		HelpTextFn: m.HelpText,
		Panel:      m,
		Panels:     panels,
		PanelIndex: panelIdx,
		Context:    ctx,
		Request:    r,
		Form:       form,
	}
}

func LabeledRowPanel(label any, helpText any, panels ...Panel) Panel {
	if pnl, ok := label.(Panel); ok {
		panels = append([]Panel{pnl}, panels...)
		label = nil
	}

	if pnl, ok := helpText.(Panel); ok {
		panels = append([]Panel{pnl}, panels...)
		helpText = nil
	}

	return &rowPanel{
		panels:   panels,
		Label:    trans.GetTextFunc(label),
		HelpText: trans.GetTextFunc(helpText),
	}
}

func RowPanel(panels ...Panel) Panel {
	return &rowPanel{
		panels: panels,
	}
}

type panelGroup struct {
	panels    []Panel
	Label     func(context.Context) string
	HelpText  func(context.Context) string
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

func (f *panelGroup) Validate(r *http.Request, ctx context.Context, form forms.Form, data map[string]any) []error {
	var errs []error
	for _, panel := range f.panels {
		if v, ok := panel.(ValidatorPanel); ok {
			errs = append(errs, v.Validate(r, ctx, form, data)...)
		}
	}
	return errs
}

func (g *panelGroup) Bind(r *http.Request, panelIdx []int, form forms.Form, ctx context.Context, boundFields map[string]forms.BoundField) BoundPanel {
	var panels = make([]BoundPanel, 0, len(g.panels))
	for _, panel := range BindPanels(g.panels, r, panelIdx, form, ctx, boundFields) {
		panels = append(panels, panel)
	}
	return &BoundPanelGroup[forms.Form]{
		LabelFn:    g.Label,
		HelpTextFn: g.HelpText,
		Panel:      g,
		Panels:     panels,
		PanelIndex: panelIdx,
		Context:    ctx,
		Request:    r,
		Form:       form,
	}
}

func LabeledPanelGroup(label any, helpText any, panels ...Panel) Panel {
	if pnl, ok := label.(Panel); ok {
		panels = append([]Panel{pnl}, panels...)
		label = nil
	}

	if pnl, ok := helpText.(Panel); ok {
		panels = append([]Panel{pnl}, panels...)
		helpText = nil
	}

	return &panelGroup{
		panels:   panels,
		Label:    trans.GetTextFunc(label),
		HelpText: trans.GetTextFunc(helpText),
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

func (a *AlertPanel) Bind(r *http.Request, panelIdx []int, form forms.Form, ctx context.Context, boundFields map[string]forms.BoundField) BoundPanel {
	return &BoundAlertPanel[forms.Form]{
		Panel:      a,
		PanelIndex: panelIdx,
		Form:       form,
		Context:    ctx,
		Request:    r,
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

var (
	_ Panel = (*tabbedPanel)(nil)
)

type tabbedPanel struct {
	className string
	tabs      []TabPanel
}

func TabbedPanel(tabs ...TabPanel) Panel {
	return &tabbedPanel{
		tabs: tabs,
	}
}

func PanelTab(title any, panels ...Panel) TabPanel {
	if pnl, ok := title.(Panel); ok {
		panels = append([]Panel{pnl}, panels...)
		title = nil
	}
	return TabPanel{
		title:  trans.GetTextFunc(title),
		panels: panels,
	}
}

type TabPanel struct {
	title  any // string | func(context.Context) string
	panels []Panel
}

func (t *tabbedPanel) Fields() []string {
	var fields []string
	for _, tab := range t.tabs {
		for _, panel := range tab.panels {
			fields = append(fields, panel.Fields()...)
		}
	}
	return fields
}

func (t *tabbedPanel) ClassName() string {
	return t.className
}

func (t *tabbedPanel) Class(classes string) Panel {
	t.className = classes
	return t
}

func (t *tabbedPanel) Validate(r *http.Request, ctx context.Context, form forms.Form, data map[string]any) []error {
	var errs []error
	for _, tab := range t.tabs {
		for _, panel := range tab.panels {
			if v, ok := panel.(ValidatorPanel); ok {
				errs = append(errs, v.Validate(r, ctx, form, data)...)
			}
		}
	}
	return errs
}

func (t *tabbedPanel) Bind(r *http.Request, panelIdx []int, form forms.Form, ctx context.Context, boundFields map[string]forms.BoundField) BoundPanel {
	var boundTabs = make([]*boundTabPanel, 0, len(t.tabs))
	for _, tab := range t.tabs {

		var boundPanels = make([]BoundPanel, 0, len(tab.panels))
		for _, panel := range BindPanels(tab.panels, r, panelIdx, form, ctx, boundFields) {
			boundPanels = append(boundPanels, panel)
		}

		boundTabs = append(boundTabs, &boundTabPanel{
			title:  trans.GetTextFunc(tab.title),
			panels: boundPanels,
		})
	}

	return &boundTabbedPanel{
		request:  r,
		panelIdx: panelIdx,
		panels:   boundTabs,
		context:  ctx,
	}
}
