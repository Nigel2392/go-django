package admin

import (
	"context"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/core/ctx"
	"github.com/Nigel2392/django/forms"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/django/forms/media"
	"github.com/Nigel2392/django/forms/modelforms"
	"github.com/Nigel2392/django/forms/widgets"
	"github.com/elliotchance/orderedmap/v2"
)

var _ forms.Form = (*AdminForm[modelforms.ModelForm[attrs.Definer]])(nil)

type Panel interface {
	Bind(form forms.Form, ctx context.Context, boundFields map[string]forms.BoundField) BoundPanel
}

type fieldPanel struct {
	fieldname string
}

func (f *fieldPanel) Bind(form forms.Form, ctx context.Context, boundFields map[string]forms.BoundField) BoundPanel {
	var bf, ok = boundFields[f.fieldname]
	if !ok {
		panic("Field does not exist in form")
	}

	return &BoundFormPanel[forms.Form, *fieldPanel]{
		Panel:      f,
		Form:       form,
		Context:    ctx,
		BoundField: bf,
	}
}

func FieldPanel(fieldname string) Panel {
	return &fieldPanel{
		fieldname: fieldname,
	}
}

type titlePanel struct {
	Panel
}

func (t *titlePanel) Bind(form forms.Form, ctx context.Context, boundFields map[string]forms.BoundField) BoundPanel {
	return &BoundTitlePanel[forms.Form, *titlePanel]{
		BoundPanel: t.Panel.Bind(form, ctx, boundFields),
		Context:    ctx,
	}
}

func TitlePanel(panel Panel) Panel {
	return &titlePanel{
		Panel: panel,
	}
}

type multiPanel struct {
	panels []Panel
	Label  func() string
}

func (m *multiPanel) Bind(form forms.Form, ctx context.Context, boundFields map[string]forms.BoundField) BoundPanel {
	var panels = make([]BoundPanel, 0)
	for _, panel := range m.panels {
		panels = append(panels, panel.Bind(form, ctx, boundFields))
	}
	return &BoundMultiPanel[forms.Form]{
		LabelFn: m.Label,
		Panels:  panels,
		Context: ctx,
		Form:    form,
	}
}

func MultiPanel(panels ...Panel) Panel {
	return &multiPanel{
		panels: panels,
	}
}

type PanelBoundForm struct {
	forms.BoundForm
	BoundPanels []BoundPanel
	Panels      []Panel
	Context     context.Context
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

type AdminForm[T modelforms.ModelForm[attrs.Definer]] struct {
	Form   T
	Panels []Panel
}

func NewAdminForm[T modelforms.ModelForm[attrs.Definer]](form T, panels ...Panel) *AdminForm[T] {
	return &AdminForm[T]{
		Form:   form,
		Panels: panels,
	}
}

func (a *AdminForm[T]) EditContext(key string, context ctx.Context) {
	context.Set(key, a.BoundForm())
}

func (a *AdminForm[T]) AsP() template.HTML {
	return a.Form.AsP()
}
func (a *AdminForm[T]) AsUL() template.HTML {
	return a.Form.AsUL()
}
func (a *AdminForm[T]) Media() media.Media {
	return a.Form.Media()
}
func (a *AdminForm[T]) Prefix() string {
	return a.Form.Prefix()
}
func (a *AdminForm[T]) SetPrefix(prefix string) {
	a.Form.SetPrefix(prefix)
}
func (a *AdminForm[T]) SetInitial(initial map[string]interface{}) {
	a.Form.SetInitial(initial)
}
func (a *AdminForm[T]) SetValidators(validators ...func(forms.Form) []error) {
	a.Form.SetValidators(validators...)
}
func (a *AdminForm[T]) Ordering(o []string) {
	a.Form.Ordering(o)
}
func (a *AdminForm[T]) FieldOrder() []string {
	return a.Form.FieldOrder()
}
func (a *AdminForm[T]) Field(name string) fields.Field {
	return a.Form.Field(name)
}
func (a *AdminForm[T]) Widget(name string) widgets.Widget {
	return a.Form.Widget(name)
}
func (a *AdminForm[T]) Fields() []fields.Field {
	return a.Form.Fields()
}
func (a *AdminForm[T]) Widgets() []widgets.Widget {
	return a.Form.Widgets()
}
func (a *AdminForm[T]) AddField(name string, field fields.Field) {
	a.Form.AddField(name, field)
}
func (a *AdminForm[T]) AddWidget(name string, widget widgets.Widget) {
	a.Form.AddWidget(name, widget)
}
func (a *AdminForm[T]) BoundForm() forms.BoundForm {
	var (
		form      = a.Form.BoundForm()
		boundForm = &PanelBoundForm{
			BoundForm:   form,
			Panels:      a.Panels,
			BoundPanels: make([]BoundPanel, 0),
			Context:     context.Background(),
		}
		boundFields = form.Fields()
		boundMap    = make(map[string]forms.BoundField)
	)

	for _, field := range boundFields {
		boundMap[field.Name()] = field
	}

	if len(a.Panels) > 0 {
		for _, panel := range a.Panels {
			var boundPanel = panel.Bind(a.Form, boundForm.Context, boundMap)
			boundForm.BoundPanels = append(
				boundForm.BoundPanels, boundPanel,
			)
		}
	} else {
		var fields []fields.Field
		if len(a.FieldOrder()) > 0 {
			for _, name := range a.FieldOrder() {
				fields = append(fields, a.Field(name))
			}
		} else {
			fields = a.Fields()
		}

		for _, field := range fields {
			var panel = FieldPanel(field.Name())
			var boundPanel = panel.Bind(a.Form, boundForm.Context, boundMap)
			boundForm.BoundPanels = append(
				boundForm.BoundPanels, boundPanel,
			)
		}
	}

	return boundForm
}
func (a *AdminForm[T]) BoundFields() *orderedmap.OrderedMap[string, forms.BoundField] {
	return a.Form.BoundFields()
}
func (a *AdminForm[T]) BoundErrors() *orderedmap.OrderedMap[string, []error] {
	return a.Form.BoundErrors()
}
func (a *AdminForm[T]) ErrorList() []error {
	return a.Form.ErrorList()
}
func (a *AdminForm[T]) WithData(data url.Values, files map[string][]io.ReadCloser, r *http.Request) forms.Form {
	return a.Form.WithData(data, files, r)
}
func (a *AdminForm[T]) CleanedData() map[string]interface{} {
	return a.Form.CleanedData()
}
func (a *AdminForm[T]) FullClean() {
	a.Form.FullClean()
}
func (a *AdminForm[T]) Validate() {
	a.Form.Validate()
}
func (a *AdminForm[T]) HasChanged() bool {
	return a.Form.HasChanged()
}
func (a *AdminForm[T]) IsValid() bool {
	return a.Form.IsValid()
}
func (a *AdminForm[T]) Close() error {
	return a.Form.Close()
}
func (a *AdminForm[T]) OnValid(f ...func(forms.Form)) {
	a.Form.OnValid(f...)
}
func (a *AdminForm[T]) OnInvalid(f ...func(forms.Form)) {
	a.Form.OnInvalid(f...)
}
func (a *AdminForm[T]) OnFinalize(f ...func(forms.Form)) {
	a.Form.OnFinalize(f...)
}
func (a *AdminForm[T]) Load() {
	a.Form.Load()
}
func (a *AdminForm[T]) Save() error {
	return a.Form.Save()
}
func (a *AdminForm[T]) WithContext(ctx context.Context) {
	a.Form.WithContext(ctx)
}
func (a *AdminForm[T]) Context() context.Context {
	return a.Form.Context()
}
func (a *AdminForm[T]) SetFields(fields ...string) {
	a.Form.SetFields(fields...)
}
func (a *AdminForm[T]) SetExclude(exclude ...string) {
	a.Form.SetExclude(exclude...)
}
func (a *AdminForm[T]) Instance() attrs.Definer {
	return a.Form.Instance()
}
func (a *AdminForm[T]) SetInstance(model attrs.Definer) {
	a.Form.SetInstance(model)
}
