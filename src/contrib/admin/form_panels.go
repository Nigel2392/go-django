package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"maps"
	"net/http"
	"reflect"
	"slices"
	"sort"
	"strings"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/fields/formfields"
	"github.com/Nigel2392/go-django/src/contrib/admin/compare"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/formsets"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/Nigel2392/go-django/src/forms/widgets/chooser"
	"github.com/a-h/templ"
	"github.com/elliotchance/orderedmap/v2"
)

func panelTypeName(panel Panel) string {
	var rT = reflect.TypeOf(panel)
	for rT.Kind() == reflect.Ptr {
		rT = rT.Elem()
	}
	return strings.ToLower(rT.Name())
}

func panelPathPart(panel Panel, suffix any) string {
	if suffix == nil {
		return panelTypeName(panel)
	}
	return fmt.Sprintf("%s-%s", panelTypeName(panel), suffix)
}

type PanelTree struct {
	Parent      *PanelTree
	ContentPath []string
	Panels      []Panel
	Nodes       *orderedmap.OrderedMap[string, *PanelTree]
}

func (p *PanelTree) FindNode(contentPath []string) *PanelTree {
	return p.findNode(contentPath, 0)
}

func (p *PanelTree) findNode(contentPath []string, index int) *PanelTree {
	if index >= len(contentPath) {
		return p
	}

	var key = contentPath[index]
	var child, ok = p.Nodes.Get(key)
	if !ok {
		return nil
	}

	return child.findNode(contentPath, index+1)
}

func newPanelTree(panels []Panel, parent *PanelTree, contentPath []string) *PanelTree {
	var root = &PanelTree{
		Panels:      panels,
		Parent:      parent,
		Nodes:       orderedmap.NewOrderedMap[string, *PanelTree](),
		ContentPath: contentPath,
	}

	for i, panel := range panels {
		var key = fmt.Sprintf(
			"%s-%d",
			panelTypeName(panel), i,
		)

		if mp, ok := panel.(MultiPanel); ok {
			var child = newPanelTree(mp.Children(), root, append(contentPath, key))
			root.Nodes.Set(key, child)
			continue
		}

		root.Nodes.Set(key, &PanelTree{
			Parent:      root,
			Panels:      []Panel{panel},
			Nodes:       orderedmap.NewOrderedMap[string, *PanelTree](),
			ContentPath: append(contentPath, key),
		})
	}

	return root
}

func NewPanelTree(panels ...Panel) *PanelTree {
	return newPanelTree(panels, nil, []string{})
}

type Panel interface {
	Fields() []string
	ClassName() string
	Class(classes string) Panel
	Comparison(ctx context.Context, oldInstance attrs.Definer, newInstance attrs.Definer) (compare.Comparison, error)
	Bind(r *http.Request, panelCount map[string]int, form forms.Form, ctx context.Context, instance attrs.Definer, boundFields map[string]forms.BoundField, formset FormSetObject) BoundPanel
}

type ValidatorPanel interface {
	Panel
	Validate(r *http.Request, ctx context.Context, form forms.Form, data map[string]any) []error
}

/*
map[string]any ( map[string]any -> recursively, []formsets.BaseFormSetForm, formsets.BaseFormSetForm)
*/
type FormSetMap map[string]FormSetObject

// This can be either map[string]any, []formsets.BaseFormSetForm or formsets.BaseFormSetForm
type FormSetObject any

type FormPanel interface {
	Panel
	Forms(r *http.Request, ctx context.Context, instance attrs.Definer) (FormSetObject, []formsets.BaseFormSetForm, error)
}

type MultiPanel interface {
	Panel
	Children() []Panel
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

func (f *fieldPanel) Bind(r *http.Request, _ map[string]int, form forms.Form, ctx context.Context, instance attrs.Definer, boundFields map[string]forms.BoundField, _ FormSetObject) BoundPanel {
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

func (t *titlePanel) Bind(r *http.Request, panelCount map[string]int, form forms.Form, ctx context.Context, instance attrs.Definer, boundFields map[string]forms.BoundField, formsets FormSetObject) BoundPanel {
	var panel = t.Panel.Bind(r, panelCount, form, ctx, instance, boundFields, formsets)
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

func (f *rowPanel) Children() []Panel {
	return f.panels
}

func (f *rowPanel) Forms(r *http.Request, ctx context.Context, instance attrs.Definer) (FormSetObject, []formsets.BaseFormSetForm, error) {
	var m FormSetMap = make(map[string]FormSetObject)
	var l = make([]formsets.BaseFormSetForm, 0)
	for i, panel := range f.panels {
		if fp, ok := panel.(FormPanel); ok {
			f, formList, err := fp.Forms(r, ctx, instance)
			if err != nil {
				logger.Errorf("rowPanel.Forms: error getting forms from panel: %v", err)
				continue
			}
			m[panelPathPart(panel, i)] = f
			l = append(l, formList...)
		}
	}
	return m, l, nil
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

func (m *rowPanel) Bind(r *http.Request, panelCount map[string]int, form forms.Form, ctx context.Context, instance attrs.Definer, boundFields map[string]forms.BoundField, formsets FormSetObject) BoundPanel {
	var panels = make([]BoundPanel, 0, len(m.panels))
	for _, panel := range BindPanels(m.panels, r, panelCount, form, ctx, instance, boundFields, formsets) {
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

func (f *panelGroup) Children() []Panel {
	return f.panels
}

func (f *panelGroup) Forms(r *http.Request, ctx context.Context, instance attrs.Definer) (FormSetObject, []formsets.BaseFormSetForm, error) {
	var fmap FormSetMap = make(map[string]FormSetObject)
	var l = make([]formsets.BaseFormSetForm, 0)
	for i, panel := range f.panels {
		if fp, ok := panel.(FormPanel); ok {
			f, formList, err := fp.Forms(r, ctx, instance)
			if err != nil {
				return nil, nil, err
			}
			fmap[panelPathPart(panel, i)] = f
			l = append(l, formList...)
		}
	}
	return fmap, l, nil
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

func (g *panelGroup) Bind(r *http.Request, panelCount map[string]int, form forms.Form, ctx context.Context, instance attrs.Definer, boundFields map[string]forms.BoundField, formsets FormSetObject) BoundPanel {
	var panels = make([]BoundPanel, 0, len(g.panels))
	for _, panel := range BindPanels(g.panels, r, panelCount, form, ctx, instance, boundFields, formsets) {
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

func (a *AlertPanel) Bind(r *http.Request, _ map[string]int, form forms.Form, ctx context.Context, _ attrs.Definer, _ map[string]forms.BoundField, _ FormSetObject) BoundPanel {
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

func (t *tabbedPanel) Children() []Panel {
	var panels = make([]Panel, 0)
	for _, tab := range t.tabs {
		panels = append(panels, tab.panels...)
	}
	return panels
}

func (t *tabbedPanel) Forms(r *http.Request, ctx context.Context, instance attrs.Definer) (FormSetObject, []formsets.BaseFormSetForm, error) {
	var fMap FormSetMap = make(map[string]FormSetObject)
	var formsList []formsets.BaseFormSetForm
	for i, tab := range t.tabs {
		for j, panel := range tab.panels {
			if fp, ok := panel.(FormPanel); ok {
				f, formList, err := fp.Forms(r, ctx, instance)
				if err != nil {
					return nil, nil, err
				}
				fMap[fmt.Sprintf("tab-%d-panel-%d", i, j)] = f
				formsList = append(formsList, formList...)
			}
		}
	}
	return fMap, formsList, nil
}

func (t *tabbedPanel) Comparison(ctx context.Context, oldInstance attrs.Definer, newInstance attrs.Definer) (compare.Comparison, error) {
	var allPanels = make([]Panel, 0)
	for _, tab := range t.tabs {
		allPanels = append(allPanels, tab.panels...)
	}
	return PanelComparison(ctx, allPanels, oldInstance, newInstance)
}

func (t *tabbedPanel) Bind(r *http.Request, panelCount map[string]int, form forms.Form, ctx context.Context, instance attrs.Definer, boundFields map[string]forms.BoundField, formsets FormSetObject) BoundPanel {
	var boundTabs = make([]*boundTabPanel, 0, len(t.tabs))
	var fMap, ok = formsets.(FormSetMap)
	assert.True(
		ok && formsets != nil || formsets == nil,
		"formsets provided to tabbedPanel.Bind are required to be of type FormSetMap, got %T",
		formsets,
	)

	for i, tab := range t.tabs {
		var boundPanels = make([]BoundPanel, 0, len(tab.panels))
		for j, panel := range tab.panels {
			var boundPanel = panel.Bind(r, panelCount, form, ctx, instance, boundFields, fMap[fmt.Sprintf("tab-%d-panel-%d", i, j)])
			if boundPanel != nil {
				boundPanels = append(boundPanels, boundPanel)
			}
		}

		if len(boundPanels) == 0 {
			continue
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

var _ FormPanel = (*ModelFormPanel[attrs.Definer, modelforms.ModelForm[attrs.Definer]])(nil)

type ModelFormPanel[TARGET attrs.Definer, FORM modelforms.ModelForm[TARGET]] struct {
	Form           func() FORM
	SaveInstance   func(ctx context.Context, form FORM, instance TARGET) error
	TargetType     TARGET
	MaxNum         int
	MinNum         int
	DisallowAdd    bool
	DisallowRemove bool
	Extra          int
	Classname      string
	SubClassname   string
	FieldName      string
	Panels         []Panel
}

func ModelPanel[TARGET attrs.Definer, FORM modelforms.ModelForm[TARGET]](fieldName string, targetType TARGET, options ...func(*ModelFormPanel[TARGET, FORM])) *ModelFormPanel[TARGET, FORM] {
	var panel = &ModelFormPanel[TARGET, FORM]{
		TargetType: targetType,
		FieldName:  fieldName,
	}
	for _, option := range options {
		option(panel)
	}
	return panel
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

func (f *ModelFormPanel[TARGET, FORM]) getField(source attrs.Definer) attrs.FieldDefinition {
	var meta = attrs.GetModelMeta(source)
	var defs = meta.Definitions()
	var field, ok = defs.Field(f.FieldName)
	except.Assert(
		ok, http.StatusInternalServerError,
		"ModelFormPanel: field %q not found in model %T",
		f.FieldName, source,
	)
	return field
}

func (p *ModelFormPanel[TARGET, FORM]) GetForms(ctx context.Context, r *http.Request, source attrs.Definer, minNumForms int, targetList []TARGET) []modelforms.ModelForm[TARGET] {
	var formList = make([]modelforms.ModelForm[TARGET], 0, minNumForms)
	var field = p.getField(source)
	var rel = field.Rel()
	var revField = rel.Field()
	for i := 0; i < max(minNumForms, len(targetList)); i++ {
		var target TARGET
		var isNew = i >= len(targetList)
		if !isNew {
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

			var opts FormViewOptions
			if isNew {
				opts = modelDef.AddView
			} else {
				opts = modelDef.EditView
			}

			form = GetAdminForm(
				target,
				opts,
				modelDef._app,
				modelDef,
				r,
			)
		}

		var initialData = make(map[string]any)
		switch field.Rel().Type() {
		case attrs.RelOneToMany:
			if revField != nil {
				var fieldName = revField.Name()
				var pk = attrs.PrimaryKey(source)
				if !fields.IsZero(pk) {
					initialData[fieldName] = pk
				} else {
					logger.Warnf("ModelFormPanel: source instance has zero primary key; related object may not be saved correctly")
				}

				form.AddField(fieldName, &formfields.ForeignKeyFormField{
					BaseRelationField: formfields.BaseRelationField{
						BaseField: fields.NewField(
							fields.Widget(formfields.ModelSelectWidget(
								false, "", chooser.BaseChooserOptions{
									TargetObject: source,
									GetPrimaryKey: func(ctx context.Context, i interface{}) interface{} {
										return attrs.PrimaryKey(i.(attrs.Definer))
									},
									Queryset: func(ctx context.Context) ([]interface{}, error) {
										return []interface{}{source}, nil
									},
								},
								nil,
							)),
							fields.Hide(true),
						),
						Field:    revField,
						Relation: revField.Rel(),
					},
				})
			} else {
				assert.Fail(
					"ModelFormPanel: reverse field not found for one-to-many relation on field %s",
					field.Name(),
				)
			}
			form = &clusterableForm[TARGET, FORM]{
				ModelForm:    form,
				FormOrdering: FORM_ORDERING_POST,
			}
		case attrs.RelManyToOne:
			form = &clusterableForm[TARGET, FORM]{
				ModelForm:    form,
				FormOrdering: FORM_ORDERING_PRE,
			}
		default:
			// not implemented
			assert.Fail(
				"ModelFormPanel: relation type %s not implemented",
				field.Rel().Type(),
			)
		}

		form.SetInstance(target)
		form.Load()

		var init = maps.Clone(form.InitialData())
		maps.Copy(init, initialData)
		form.SetInitial(init)

		formList = append(formList, form)
	}
	return formList
}

var _ ClusterOrderableForm = (*clusterableForm[attrs.Definer, modelforms.ModelForm[attrs.Definer]])(nil)

type clusterableForm[T1 attrs.Definer, T2 modelforms.ModelForm[T1]] struct {
	modelforms.ModelForm[T1]
	FormOrdering FORM_ORDERING
}

func (c *clusterableForm[T1, T2]) FormOrder() FORM_ORDERING {
	return c.FormOrdering
}

func (c *clusterableForm[T1, T2]) Unwrap() []any {
	if unwrapper, ok := c.ModelForm.(forms.FormWrapper[any]); ok {
		return unwrapper.Unwrap()
	}
	if unwrapper, ok := c.ModelForm.(forms.FormWrapper[T1]); ok {
		var unwrapped = unwrapper.Unwrap()
		var list = make([]any, len(unwrapped))
		for i, item := range unwrapped {
			list[i] = item
		}
		return list
	}
	return []any{c.ModelForm}
}

func (m *ModelFormPanel[TARGET, FORM]) Forms(r *http.Request, ctx context.Context, instance attrs.Definer) (FormSetObject, []formsets.BaseFormSetForm, error) {
	var f = m.FormSet(r, ctx, instance)
	return f, []formsets.BaseFormSetForm{f}, nil
}

func (m *ModelFormPanel[TARGET, FORM]) FormSet(r *http.Request, ctx context.Context, instance attrs.Definer) formsets.ListFormSet[modelforms.ModelForm[TARGET]] {
	var f = formsets.NewBaseFormSet(
		ctx, formsets.FormsetOptions[modelforms.ModelForm[TARGET]]{
			MinNum:     m.MinNum,
			MaxNum:     m.MaxNum,
			Extra:      m.Extra,
			CanDelete:  !m.DisallowRemove,
			CanAdd:     !m.DisallowAdd,
			CanOrder:   true,
			HideDelete: true,
			NewForm: func(c context.Context) modelforms.ModelForm[TARGET] {
				var forms = m.GetForms(ctx, r, instance, 1, nil)
				return forms[0]
			},
			DefaultForms: func(ctx context.Context, max, min int) ([]modelforms.ModelForm[TARGET], error) {
				var _, targetList, err = getRelatedList[TARGET](r, ctx, instance, m.FieldName)
				if err != nil {
					return nil, err
				}

				return m.GetForms(ctx, r, instance, min, targetList), nil
			},
		},
	)
	f.SetPrefix(m.FieldName)
	return f
}

func getRelatedList[TARGET attrs.Definer](r *http.Request, ctx context.Context, source attrs.Definer, fieldName string) (attrs.Field, []TARGET, error) {
	var defs = source.FieldDefs()
	field, ok := defs.Field(fieldName)
	if !ok {
		return nil, nil, errors.FieldNotFound.Wrapf(
			"Field %q not found in instance of type %T", fieldName, source,
		)
	}

	var rel = field.Rel()
	if rel.Type() == attrs.RelOneToMany {
		var qs = queries.OneToManyQuerySet[TARGET](&queries.RelRevFK[attrs.Definer]{
			Parent: &queries.ParentInfo{
				Object: source,
				Field:  field,
			},
		})

		var rows, err = qs.All()
		if err != nil {
			return nil, nil, errors.Wrapf(
				err, "Error fetching related objects for field %q", fieldName,
			)
		}

		var list = make([]TARGET, len(rows))
		for i, row := range rows {
			list[i] = row.Object
		}

		return field, list, nil
	}

	var _null TARGET
	var value = field.GetValue()
	var targetList = make([]TARGET, 0)
	switch v := value.(type) {
	case []TARGET:
		targetList = v
	case []attrs.Definer:
		for _, item := range v {
			t, ok := item.(TARGET)
			if !ok {
				return nil, nil, errors.TypeMismatch.Wrapf(
					"Item of type %T in field %q is not of expected type %T",
					item, fieldName, _null,
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
				return nil, nil, errors.TypeMismatch.Wrapf(
					"Item of type %T in field %q is not of expected type %T",
					item, fieldName, _null,
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
	return field, targetList, nil
}

func (m *ModelFormPanel[TARGET, FORM]) validate(rel attrs.Relation, base int, list []TARGET) (total int, canAdd bool, targets []TARGET) {
	var minNumForms = m.MinNum
	var maxNumForms = m.MaxNum
	var totalForms = min(base+m.Extra, maxNumForms)
	if minNumForms > 0 && totalForms < minNumForms {
		totalForms = minNumForms
	}

	switch rel.Type() {
	case attrs.RelManyToOne, attrs.RelOneToOne:
		if m.MinNum > 1 {
			m.MinNum = 1
			logger.Warnf("ModelFormPanel.Bind: MinNum > 1 for ManyToOne or OneToOne relation; setting MinNum to 1")
		}

		if m.MaxNum > 1 {
			m.MaxNum = 1
			logger.Warnf("ModelFormPanel.Bind: MaxNum > 1 for ManyToOne or OneToOne relation; setting MaxNum to 1")
		}

		if m.Extra > 1 {
			m.Extra = 0
			logger.Warnf("ModelFormPanel.Bind: Extra > 0 for ManyToOne or OneToOne relation; setting Extra to 0")
		}

		if len(list) > 1 {
			logger.Warnf("ModelFormPanel.Bind: more than one related object found for ManyToOne or OneToOne relation; only the first will be used")
			list = list[:1]
		}
	}

	var canAddMore = maxNumForms == 0 || totalForms < maxNumForms
	return totalForms, canAddMore, list
}

func (m *ModelFormPanel[TARGET, FORM]) Bind(r *http.Request, panelCount map[string]int, form forms.Form, ctx context.Context, instance attrs.Definer, boundFields map[string]forms.BoundField, formSets FormSetObject) BoundPanel {
	var field, targetList, err = getRelatedList[TARGET](r, ctx, instance, m.FieldName)
	if err != nil {
		except.Fail(
			http.StatusInternalServerError,
			"ModelFormPanel.Bind: %T / %v", instance, err,
		)
	}

	_, _, targetList = m.validate(
		field.Rel(),
		len(targetList),
		targetList,
	)

	var panels = make([]Panel, 0, len(m.Panels)+1)
	var seen = make(map[string]struct{})
	for _, panel := range m.Panels {
		for _, fname := range panel.Fields() {
			seen[fname] = struct{}{}
		}
		panels = append(panels, panel)
	}

	var fld = m.getField(instance)
	var relFld = fld.Rel().Field()
	if relFld != nil {
		if _, ok := seen[relFld.Name()]; !ok {
			panels = append(panels, FieldPanel(relFld.Name()))
			seen[relFld.Name()] = struct{}{}
		}
	}

	var formset formsets.ListFormSet[modelforms.ModelForm[TARGET]]
	if typ, ok := formSets.(formsets.ListFormSet[modelforms.ModelForm[TARGET]]); ok {
		formset = typ
	} else {
		formset = m.FormSet(r, ctx, instance)
	}

	//	var meta = attrs.GetModelMeta(fld.Rel().Model())
	//	var defs = meta.Definitions()
	//	var primary = defs.Primary()
	var emptyForm = formset.NewForm(ctx)
	var fieldMap = emptyForm.BoundFields()
	var keys = fieldMap.Keys()
	if len(panels) != len(keys) {
		for _, key := range keys {
			if _, ok := seen[key]; ok {
				continue
			}
			panels = append(panels, FieldPanel(key))
			seen[key] = struct{}{}
		}
	}

	if _, ok := seen[formsets.DELETION_FIELD_NAME]; !ok && !m.DisallowRemove {
		panels = append(panels, FieldPanel(formsets.DELETION_FIELD_NAME))
		seen[formsets.DELETION_FIELD_NAME] = struct{}{}
	}

	if _, ok := seen[formsets.ORDERING_FIELD_NAME]; !ok && !m.DisallowAdd {
		panels = append(panels, FieldPanel(formsets.ORDERING_FIELD_NAME))
		seen[formsets.ORDERING_FIELD_NAME] = struct{}{}
	}

	return &BoundModelFormPanel[TARGET, FORM]{
		Panel:       m,
		Panels:      panels,
		FormSet:     formset,
		SourceModel: instance,
		SourceField: field,
		Context:     ctx,
		Request:     r,
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

func (j JSONDetailPanel) Bind(r *http.Request, _ map[string]int, form forms.Form, _ context.Context, _ attrs.Definer, boundFieldsMap map[string]forms.BoundField, _ FormSetObject) BoundPanel {
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
