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

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/src/contrib/admin/compare"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/a-h/templ"
)

type Panel interface {
	Fields() []string
	ClassName() string
	Class(classes string) Panel
	Comparison(ctx context.Context, oldInstance attrs.Definer, newInstance attrs.Definer) (compare.Comparison, error)
	Bind(r *http.Request, panelCount map[string]int, form forms.Form, ctx context.Context, instance attrs.Definer, boundFields map[string]forms.BoundField) BoundPanel
}

type ValidatorPanel interface {
	Panel
	Validate(r *http.Request, ctx context.Context, form forms.Form, data map[string]any) []error
}

type FormPanel interface {
	Panel
	Forms() []forms.Form
}

type BoundPanel interface {
	Hidden() bool
	Component() templ.Component
	Render() template.HTML
}

func PanelComparison(ctx context.Context, panels []Panel, oldInstance attrs.Definer, newInstance attrs.Definer, mustWrap ...bool) (compare.Comparison, error) {
	var comparisons = make([]compare.Comparison, 0, len(panels))
	for _, panel := range panels {
		var comp, err = panel.Comparison(ctx, oldInstance, newInstance)
		if err != nil {
			return nil, err
		}

		if comp == nil {
			continue
		}

		if unwrapper, ok := comp.(compare.ComparisonWrapper); ok {
			comparisons = append(comparisons, unwrapper.Unwrap()...)
		} else {
			comparisons = append(comparisons, comp)
		}
	}

	if len(comparisons) == 1 && (len(mustWrap) == 0 || !mustWrap[0]) {
		return comparisons[0], nil
	}

	if len(comparisons) == 0 {
		return nil, nil
	}

	return compare.MultipleComparison(ctx, comparisons...), nil
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

func (f *fieldPanel) Comparison(ctx context.Context, oldInstance attrs.Definer, newInstance attrs.Definer) (compare.Comparison, error) {
	return compare.GetComparison(
		ctx,
		nil,
		nil,
		f.fieldname,
		oldInstance,
		newInstance,
	)
}

func (f *fieldPanel) Bind(r *http.Request, _ map[string]int, form forms.Form, ctx context.Context, instance attrs.Definer, boundFields map[string]forms.BoundField) BoundPanel {
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

func (t *titlePanel) Bind(r *http.Request, panelCount map[string]int, form forms.Form, ctx context.Context, instance attrs.Definer, boundFields map[string]forms.BoundField) BoundPanel {
	var panel = t.Panel.Bind(r, panelCount, form, ctx, instance, boundFields)
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

func (f *rowPanel) Forms() []forms.Form {
	var formsList []forms.Form
	for _, panel := range f.panels {
		if fp, ok := panel.(FormPanel); ok {
			formsList = append(formsList, fp.Forms()...)
		}
	}
	return formsList
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

func (m *rowPanel) Bind(r *http.Request, panelCount map[string]int, form forms.Form, ctx context.Context, instance attrs.Definer, boundFields map[string]forms.BoundField) BoundPanel {
	var panels = make([]BoundPanel, 0, len(m.panels))
	for _, panel := range BindPanels(m.panels, r, panelCount, form, ctx, instance, boundFields) {
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

func (f *rowPanel) Comparison(ctx context.Context, oldInstance attrs.Definer, newInstance attrs.Definer) (compare.Comparison, error) {
	return PanelComparison(ctx, f.panels, oldInstance, newInstance)
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

func (f *panelGroup) Forms() []forms.Form {
	var formsList []forms.Form
	for _, panel := range f.panels {
		if fp, ok := panel.(FormPanel); ok {
			formsList = append(formsList, fp.Forms()...)
		}
	}
	return formsList
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

func (g *panelGroup) Bind(r *http.Request, panelCount map[string]int, form forms.Form, ctx context.Context, instance attrs.Definer, boundFields map[string]forms.BoundField) BoundPanel {
	var panels = make([]BoundPanel, 0, len(g.panels))
	for _, panel := range BindPanels(g.panels, r, panelCount, form, ctx, instance, boundFields) {
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

func (f *panelGroup) Comparison(ctx context.Context, oldInstance attrs.Definer, newInstance attrs.Definer) (compare.Comparison, error) {
	return PanelComparison(ctx, f.panels, oldInstance, newInstance)
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

func (f *AlertPanel) Comparison(ctx context.Context, oldInstance attrs.Definer, newInstance attrs.Definer) (compare.Comparison, error) {
	return nil, nil
}

func (a *AlertPanel) Bind(r *http.Request, _ map[string]int, form forms.Form, ctx context.Context, _ attrs.Definer, _ map[string]forms.BoundField) BoundPanel {
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

func (t *tabbedPanel) Comparison(ctx context.Context, oldInstance attrs.Definer, newInstance attrs.Definer) (compare.Comparison, error) {
	var allPanels = make([]Panel, 0)
	for _, tab := range t.tabs {
		allPanels = append(allPanels, tab.panels...)
	}
	return PanelComparison(ctx, allPanels, oldInstance, newInstance)
}

func (t *tabbedPanel) Bind(r *http.Request, panelCount map[string]int, form forms.Form, ctx context.Context, instance attrs.Definer, boundFields map[string]forms.BoundField) BoundPanel {
	var boundTabs = make([]*boundTabPanel, 0, len(t.tabs))
	for _, tab := range t.tabs {

		var boundPanels = make([]BoundPanel, 0, len(tab.panels))
		for _, panel := range BindPanels(tab.panels, r, panelCount, form, ctx, instance, boundFields) {
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

type ModelFormPanel[TARGET attrs.Definer, FORM modelforms.ModelForm[TARGET]] struct {
	Form       func() FORM
	TargetType TARGET
	MaxNum     int
	MinNum     int
	Extra      int
	Classname  string
	FieldName  string
	Panels     []Panel
}

func (m *ModelFormPanel[TARGET, FORM]) Fields() []string {
	return []string{m.FieldName}
}

func (m *ModelFormPanel[TARGET, FORM]) ClassName() string {
	return m.Classname
}

func (m *ModelFormPanel[TARGET, FORM]) Class(classes string) Panel {
	m.Classname = classes
	return m
}

func (f *ModelFormPanel[TARGET, FORM]) Comparison(ctx context.Context, oldInstance attrs.Definer, newInstance attrs.Definer) (compare.Comparison, error) {
	return nil, nil
}

func (p *ModelFormPanel[TARGET, FORM]) formPrefix(index any) string {
	return fmt.Sprintf("%s-%v", p.FieldName, index)
}

func (p *ModelFormPanel[TARGET, FORM]) GetForms(ctx context.Context, r *http.Request, source attrs.Definer, totalForms int, targetList []TARGET) []modelforms.ModelForm[TARGET] {
	var forms = make([]modelforms.ModelForm[TARGET], 0, totalForms)
	for i := 0; i < totalForms; i++ {
		var target TARGET
		if i < len(targetList) {
			target = targetList[i]
		} else {
			target = attrs.NewObject[TARGET](p.TargetType)
		}

		var form modelforms.ModelForm[TARGET]
		if p.Form != nil {
			form = p.Form()
		} else {
			var modelDef = FindDefinition(p.TargetType)
			except.Assert(
				modelDef != nil,
				http.StatusInternalServerError,
				"ModelFormPanel: no model definition found for type %T",
				p.TargetType,
			)

			form = GetAdminForm(
				target,
				modelDef.AddView,
				modelDef._app,
				modelDef,
				r,
			)
		}

		var meta = attrs.GetModelMeta(source)
		var defs = meta.Definitions()
		var field, ok = defs.Field(p.FieldName)
		except.Assert(
			ok, http.StatusInternalServerError,
			"ModelFormPanel: field %q not found in model %T",
			p.FieldName, source,
		)

		var revField = field.Rel().Field()
		var initialData = make(map[string]any)
		if revField != nil {
			var fieldName = revField.Name()
			var pk = attrs.PrimaryKey(source)
			if !fields.IsZero(pk) {
				initialData[fieldName] = pk
			}
		}

		form.SetInstance(target)
		form.SetPrefix(
			p.formPrefix(i),
		)

		form.Load()

		form.SetInitial(initialData)

		forms = append(forms, form)
	}
	return forms
}

func (m *ModelFormPanel[TARGET, FORM]) EmptyForm(ctx context.Context, r *http.Request, source attrs.Definer) modelforms.ModelForm[TARGET] {
	var forms = m.GetForms(ctx, r, source, 1, nil)
	var f = forms[0]
	f.SetPrefix("__PREFIX__")
	return f
}

func (m *ModelFormPanel[TARGET, FORM]) Bind(r *http.Request, panelCount map[string]int, form forms.Form, ctx context.Context, instance attrs.Definer, boundFields map[string]forms.BoundField) BoundPanel {
	var defs = instance.FieldDefs()
	field, ok := defs.Field(m.FieldName)
	if !ok {
		except.Fail(
			http.StatusInternalServerError,
			fmt.Sprintf("Field %q not found in instance of type %T", m.FieldName, instance),
		)
	}

	var value = field.GetValue()
	var targetList = make([]TARGET, 0)
	switch v := value.(type) {
	case []TARGET:
		targetList = v
	case []attrs.Definer:
		for _, item := range v {
			t, ok := item.(TARGET)
			if !ok {
				except.Fail(
					http.StatusInternalServerError,
					fmt.Sprintf("Item of type %T in field %q is not of expected type %T", item, m.FieldName, m.TargetType),
				)
			}
			targetList = append(targetList, t)
		}
	case queries.RelationValue:
		targetList = []TARGET{v.GetValue().(TARGET)}
	case queries.ThroughRelationValue:
		t, _ := v.GetValue()
		targetList = []TARGET{t.(TARGET)}
	case queries.MultiRelationValue:
		for _, item := range v.GetValues() {
			t, ok := item.(TARGET)
			if !ok {
				except.Fail(
					http.StatusInternalServerError,
					fmt.Sprintf("Item of type %T in field %q is not of expected type %T", item, m.FieldName, m.TargetType),
				)
			}
			targetList = append(targetList, t)
		}
	case queries.MultiThroughRelationValue:
		for _, item := range v.GetValues() {
			t := item.Model()
			targetList = append(targetList, t.(TARGET))
		}
	}

	var minNumForms = m.MaxNum
	var maxNumForms = m.MinNum
	var totalForms = len(targetList) + m.Extra
	if minNumForms > 0 && totalForms < minNumForms {
		totalForms = minNumForms
	}

	var canAddMore = maxNumForms == 0 || totalForms < maxNumForms
	var forms = m.GetForms(ctx, r, instance, totalForms, targetList)
	if !canAddMore && len(forms) > minNumForms {
		forms = forms[:minNumForms]
	}

	return &BoundModelFormPanel[TARGET, FORM]{
		Panel:       m,
		Forms:       forms,
		SourceModel: instance,
		SourceField: field,
		Context:     ctx,
		Request:     r,
		emptyForm:   m.EmptyForm(ctx, r, instance),
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

// JSONDetailPanel does not support comparisons - it is read-only
func (f JSONDetailPanel) Comparison(ctx context.Context, oldInstance attrs.Definer, newInstance attrs.Definer) (compare.Comparison, error) {
	return nil, nil
}

func (j JSONDetailPanel) Bind(r *http.Request, _ map[string]int, form forms.Form, _ context.Context, _ attrs.Definer, boundFieldsMap map[string]forms.BoundField) BoundPanel {
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
