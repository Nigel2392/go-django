package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"maps"
	"net/http"
	"slices"
	"sort"
	"strings"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/a-h/templ"
)

type Panel interface {
	Fields() []string
	ClassName() string
	Class(classes string) Panel
	Bind(r *http.Request, panelCount map[string]int, form forms.Form, ctx context.Context, boundFields map[string]forms.BoundField) BoundPanel
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

func (f *fieldPanel) Bind(r *http.Request, _ map[string]int, form forms.Form, ctx context.Context, boundFields map[string]forms.BoundField) BoundPanel {
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
	classname    string
	outputFields []string
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

func (t *titlePanel) WithOutputFields(fields ...string) Panel {
	t.outputFields = fields
	return t
}

func (f *titlePanel) Validate(r *http.Request, ctx context.Context, form forms.Form, data map[string]any) []error {
	if v, ok := f.Panel.(ValidatorPanel); ok {
		return v.Validate(r, ctx, form, data)
	}
	return nil
}

func (t *titlePanel) Bind(r *http.Request, panelCount map[string]int, form forms.Form, ctx context.Context, boundFields map[string]forms.BoundField) BoundPanel {
	var panel = t.Panel.Bind(r, panelCount, form, ctx, boundFields)
	if panel == nil {
		return nil
	}

	var outputIds []string
	for _, field := range t.outputFields {
		var bf, ok = boundFields[field]
		if !ok {
			except.Fail(
				http.StatusInternalServerError,
				fmt.Sprintf("Field %s not found in bound fields: %v", field, boundFields),
			)
			continue
		}

		outputIds = append(outputIds, bf.ID())
	}

	return &BoundTitlePanel[forms.Form, *titlePanel]{
		Panel:      t,
		BoundPanel: panel,
		Context:    ctx,
		Request:    r,
		OutputIds:  outputIds,
	}
}

type OutputtableTitlePanel interface {
	Panel
	WithOutputFields(...string) Panel
}

func TitlePanel(panel Panel, classname ...string) OutputtableTitlePanel {
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

func (m *rowPanel) Bind(r *http.Request, panelCount map[string]int, form forms.Form, ctx context.Context, boundFields map[string]forms.BoundField) BoundPanel {
	var panels = make([]BoundPanel, 0, len(m.panels))
	for _, panel := range BindPanels(m.panels, r, panelCount, form, ctx, boundFields) {
		panels = append(panels, panel)
	}
	return &BoundRowPanel[forms.Form]{
		LabelFn:    m.Label,
		HelpTextFn: m.HelpText,
		Panel:      m,
		Panels:     panels,
		PanelIndex: panelCount,
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

func (g *panelGroup) Bind(r *http.Request, panelCount map[string]int, form forms.Form, ctx context.Context, boundFields map[string]forms.BoundField) BoundPanel {
	var panels = make([]BoundPanel, 0, len(g.panels))
	for _, panel := range BindPanels(g.panels, r, panelCount, form, ctx, boundFields) {
		panels = append(panels, panel)
	}
	return &BoundPanelGroup[forms.Form]{
		LabelFn:    g.Label,
		HelpTextFn: g.HelpText,
		Panel:      g,
		Panels:     panels,
		PanelIndex: panelCount,
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

func (a *AlertPanel) Bind(r *http.Request, _ map[string]int, form forms.Form, ctx context.Context, _ map[string]forms.BoundField) BoundPanel {
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

func (t *tabbedPanel) Bind(r *http.Request, panelCount map[string]int, form forms.Form, ctx context.Context, boundFields map[string]forms.BoundField) BoundPanel {
	var boundTabs = make([]*boundTabPanel, 0, len(t.tabs))
	for _, tab := range t.tabs {

		var boundPanels = make([]BoundPanel, 0, len(tab.panels))
		for _, panel := range BindPanels(tab.panels, r, panelCount, form, ctx, boundFields) {
			boundPanels = append(boundPanels, panel)
		}

		boundTabs = append(boundTabs, &boundTabPanel{
			title:  trans.GetTextFunc(tab.title),
			panels: boundPanels,
		})
	}

	return &boundTabbedPanel{
		request: r,
		panels:  boundTabs,
		context: ctx,
	}
}

type JSONDetailPanel struct {
	FieldName string
	Classname string
	Ordering  func(r *http.Request, fields map[string]forms.BoundField) []string
	Labels    func(r *http.Request, fields map[string]forms.BoundField) map[string]any
	Widgets   func(r *http.Request, fields map[string]forms.BoundField) map[string]widgets.Widget
}

func (j JSONDetailPanel) Fields() []string {
	return []string{j.FieldName}
}

func (j JSONDetailPanel) ClassName() string {
	return j.Classname
}

func (j JSONDetailPanel) Class(classes string) Panel {
	j.Classname = classes
	return j
}

func (j JSONDetailPanel) Bind(r *http.Request, panelCount map[string]int, form forms.Form, ctx context.Context, boundFieldsMap map[string]forms.BoundField) BoundPanel {
	var dataField, ok = boundFieldsMap[j.FieldName]
	if !ok {
		assert.Fail(
			"Field %q not found in bound fields",
			j.FieldName,
		)
		return nil
	}

	var value = dataField.Value()
	var jsonData string
	switch v := value.(type) {
	case string:
		jsonData = v
	case []byte:
		jsonData = string(v)
	default:
		assert.Fail(
			"Field %q has unsupported type %T",
			j.FieldName,
			v,
		)
	}

	var data = make(map[string]interface{})
	var err = json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		return &BoundJSONDetailPanel{
			Error: err,
		}
	}

	var widgetsMap map[string]widgets.Widget
	if j.Widgets != nil {
		widgetsMap = j.Widgets(r, boundFieldsMap)
	}

	var labelsMap map[string]any
	if j.Labels != nil {
		labelsMap = j.Labels(r, boundFieldsMap)
	}

	var fieldOrdering []string
	if j.Ordering != nil {
		fieldOrdering = j.Ordering(r, boundFieldsMap)
	}

	var boundFields = make([]forms.BoundField, 0, len(data))
	var keys = slices.Collect(maps.Keys(data))
	sort.Strings(keys)

	var ordering = make(map[string]int, len(keys))
	for i, key := range fieldOrdering {
		ordering[key] = i
	}

	slices.SortStableFunc(keys, func(i, j string) int {
		var orderI, okI = ordering[i]
		var orderJ, okJ = ordering[j]
		if okI && okJ {
			return orderI - orderJ
		} else if okI {
			return -1
		} else if okJ {
			return 1
		}
		return strings.Compare(i, j)
	})

	for _, key := range keys {
		var val = data[key]

		var label, ok = labelsMap[key]
		if !ok {
			label = key
		}

		var field = fields.NewField(
			fields.Name(key),
			fields.Label(label),
			fields.ReadOnly(true),
		)
		widget, ok := widgetsMap[key]
		if !ok {
			widget = widgets.NewTextInput(nil)
		}

		widget.SetAttrs(map[string]string{
			"readonly":   "readonly",
			"data-field": key,
		})

		boundFields = append(boundFields, forms.NewBoundFormField(
			form.Context(), widget, field, fmt.Sprintf("%s__%s", dataField.Name(), key), val, []error{},
			true,
		))
	}

	return &BoundJSONDetailPanel{
		Panel:       j,
		Request:     r,
		BoundField:  dataField,
		BoundFields: boundFields,
		Error:       err,
	}
}
