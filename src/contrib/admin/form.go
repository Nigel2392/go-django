package admin

import (
	"context"
	"html/template"
	"net/http"
	"net/url"

	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/elliotchance/orderedmap/v2"
)

var (
	_ forms.Form                          = (*AdminForm[modelforms.ModelForm[attrs.Definer]])(nil)
	_ modelforms.ModelForm[attrs.Definer] = (*AdminModelForm[modelforms.ModelForm[attrs.Definer], attrs.Definer])(nil)
)

type AdminForm[T forms.Form] struct {
	Form    T
	Panels  []Panel
	Request *http.Request
}

func NewAdminForm[T forms.Form](form T, panels ...Panel) *AdminForm[T] {
	return &AdminForm[T]{
		Form:   form,
		Panels: panels,
	}
}

func (a *AdminForm[T]) EditContext(key string, context ctx.Context) {
	context.Set(key, a.BoundForm())
}

func (a *AdminForm[T]) Context() context.Context {
	return a.Form.Context()
}

func (a *AdminForm[T]) WithContext(ctx context.Context) {
	a.Form.WithContext(ctx)
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
func (a *AdminForm[T]) SetValidators(validators ...func(forms.Form, map[string]interface{}) []error) {
	a.Form.SetValidators(validators...)
}
func (a *AdminForm[T]) Ordering(o []string) {
	a.Form.Ordering(o)
}
func (a *AdminForm[T]) FieldOrder() []string {
	return a.Form.FieldOrder()
}
func (a *AdminForm[T]) Field(name string) (fields.Field, bool) {
	return a.Form.Field(name)
}
func (a *AdminForm[T]) Widget(name string) (widgets.Widget, bool) {
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
func (a *AdminForm[T]) DeleteField(name string) bool {
	return a.Form.DeleteField(name)
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
			Context:     a.Context(),
		}
		boundFields = form.Fields()
		boundMap    = make(map[string]forms.BoundField)
	)

	for _, field := range boundFields {
		boundMap[field.Name()] = field
	}

	if len(a.Panels) > 0 {
		for _, panel := range BindPanels(a.Panels, a.Request, make(map[string]int), a, boundForm.Context, boundMap) {
			boundForm.BoundPanels = append(
				boundForm.BoundPanels, panel,
			)
		}
	} else {
		var fields []fields.Field
		if len(a.FieldOrder()) > 0 {
			for _, name := range a.FieldOrder() {
				var f, _ = a.Field(name)
				fields = append(fields, f)
			}
		} else {
			fields = a.Fields()
		}

		var idx int
		for _, field := range fields {
			var panel = FieldPanel(field.Name())
			var boundPanel = panel.Bind(a.Request, make(map[string]int), a, boundForm.Context, boundMap)
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
func (a *AdminForm[T]) BoundFields() *orderedmap.OrderedMap[string, forms.BoundField] {
	return a.Form.BoundFields()
}
func (a *AdminForm[T]) BoundErrors() *orderedmap.OrderedMap[string, []error] {
	return a.Form.BoundErrors()
}
func (a *AdminForm[T]) ErrorList() []error {
	return a.Form.ErrorList()
}
func (a *AdminForm[T]) WithData(data url.Values, files map[string][]filesystem.FileHeader, r *http.Request) forms.Form {
	a.Request = r
	return a.Form.WithData(data, files, r)
}
func (a *AdminForm[T]) InitialData() map[string]interface{} {
	return a.Form.InitialData()
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
	var (
		fields  = make([]string, 0)
		fieldsM = make(map[string]struct{})
		initial = a.Form.InitialData()
		cleaned = a.Form.CleanedData()
	)

	for _, panel := range a.Panels {
		for _, field := range panel.Fields() {
			if _, ok := fieldsM[field]; !ok {
				fields = append(fields, field)
				fieldsM[field] = struct{}{}
			}
		}
	}

	for _, fieldName := range fields {
		var f, _ = a.Form.Field(fieldName)
		if !f.ReadOnly() && f.HasChanged(initial[fieldName], cleaned[fieldName]) {
			return true
		}
	}

	return false

}
func (a *AdminForm[T]) IsValid() bool {
	return a.Form.IsValid()
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

func (a *AdminForm[T]) AddFormError(errorList ...error) {
	any(a.Form).(forms.ErrorAdder).AddFormError(errorList...)
}

func (a *AdminForm[T]) AddError(name string, errorList ...error) {
	any(a.Form).(forms.ErrorAdder).AddError(name, errorList...)
}

type AdminModelForm[T1 modelforms.ModelForm[T2], T2 attrs.Definer] struct {
	*AdminForm[T1]
}

func NewAdminModelForm[T1 modelforms.ModelForm[T2], T2 attrs.Definer](form T1, panels ...Panel) *AdminModelForm[T1, T2] {
	return &AdminModelForm[T1, T2]{
		AdminForm: NewAdminForm(form, panels...),
	}
}

func (a *AdminModelForm[T1, T2]) SetOnLoad(fn func(model T2, initialData map[string]interface{})) {
	a.Form.SetOnLoad(fn)
}

func (a *AdminModelForm[T1, T2]) Load() {
	a.Form.Load()

	var fields = a.Fields()
	if len(a.Panels) == 0 {
		a.Panels = make([]Panel, 0, len(fields))
		for _, field := range fields {
			a.Panels = append(
				a.Panels,
				FieldPanel(field.Name()),
			)
		}
		return
	}

	var panelFields = make(map[string]struct{})
	for _, panel := range a.Panels {
		for _, field := range panel.Fields() {
			panelFields[field] = struct{}{}
		}
	}

	for _, field := range fields {
		var fName = field.Name()
		if _, ok := panelFields[fName]; !ok {
			a.Form.DeleteField(fName)
		}
	}

	var validatorPanels = make([]ValidatorPanel, 0, len(a.Panels))
	for _, panel := range a.Panels {
		if v, ok := panel.(ValidatorPanel); ok {
			validatorPanels = append(validatorPanels, v)
		}
	}

	if len(validatorPanels) > 0 {
		a.SetValidators(func(f forms.Form, m map[string]interface{}) []error {
			var errors = make([]error, 0)
			for _, v := range validatorPanels {
				errors = append(errors, v.Validate(a.Request, a.Context(), a, m)...)
			}
			return errors
		})
	}
}

func (a *AdminModelForm[T1, T2]) Save() (map[string]interface{}, error) {
	return a.Form.Save()
}
func (a *AdminModelForm[T1, T2]) WithContext(ctx context.Context) {
	a.Form.WithContext(ctx)
}
func (a *AdminModelForm[T1, T2]) Context() context.Context {
	return a.Form.Context()
}
func (a *AdminModelForm[T1, T2]) SetFields(fields ...string) {
	a.Form.SetFields(fields...)
}
func (a *AdminModelForm[T1, T2]) SetExclude(exclude ...string) {
	a.Form.SetExclude(exclude...)
}
func (a *AdminModelForm[T1, T2]) Instance() attrs.Definer {
	return a.Form.Instance()
}
func (a *AdminModelForm[T1, T2]) SetInstance(model T2) {
	a.Form.SetInstance(model)
}
