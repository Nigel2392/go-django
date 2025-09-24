package admin

import (
	"context"
	"html/template"
	"net/http"
	"net/url"
	"slices"

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
	_ forms.Form                          = (*AdminForm[modelforms.ModelForm[attrs.Definer], attrs.Definer])(nil)
	_ modelforms.ModelForm[attrs.Definer] = (*AdminForm[modelforms.ModelForm[attrs.Definer], attrs.Definer])(nil)
)

type FORM_ORDERING int

const (
	FORM_ORDERING_PRE  FORM_ORDERING = -1
	FORM_ORDERING_NONE FORM_ORDERING = 0
	FORM_ORDERING_POST FORM_ORDERING = 1
)

type ClusterOrderableForm interface {
	forms.Form
	FormOrder() FORM_ORDERING
}

type AdminForm[T1 modelforms.ModelForm[T2], T2 attrs.Definer] struct {
	Form    T1
	Panels  []Panel
	Request *http.Request
}

func NewAdminForm[T1 modelforms.ModelForm[T2], T2 attrs.Definer](form T1, panels ...Panel) *AdminForm[T1, T2] {
	return &AdminForm[T1, T2]{
		Form:   form,
		Panels: panels,
	}
}

func (a *AdminForm[T1, T2]) Unwrap() []forms.Form {
	var formsList = []forms.Form{a.Form}
	if unwrapper, ok := any(a.Form).(forms.FormWrapper); ok {
		formsList = append(formsList, unwrapper.Unwrap()...)
	}

	for _, panel := range a.Panels {
		if unwrapper, ok := panel.(FormPanel); ok {
			formsList = append(formsList, unwrapper.Forms()...)
		}
	}

	slices.SortStableFunc(formsList, func(i, j forms.Form) int {
		var formA, okA = i.(ClusterOrderableForm)
		var formB, okB = j.(ClusterOrderableForm)
		if okA && okB {
			return int(formA.FormOrder()) - int(formB.FormOrder())
		} else if okA {
			return int(formA.FormOrder())
		} else if okB {
			return -int(formB.FormOrder())
		}
		return int(FORM_ORDERING_NONE)
	})

	return formsList
}

func (a *AdminForm[T1, T2]) EditContext(key string, context ctx.Context) {
	context.Set(key, a.BoundForm())
}

func (a *AdminForm[T1, T2]) Context() context.Context {
	return a.Form.Context()
}

func (a *AdminForm[T1, T2]) WithContext(ctx context.Context) {
	a.Form.WithContext(ctx)
}

func (a *AdminForm[T1, T2]) AsP() template.HTML {
	return a.Form.AsP()
}
func (a *AdminForm[T1, T2]) AsUL() template.HTML {
	return a.Form.AsUL()
}
func (a *AdminForm[T1, T2]) Media() media.Media {
	return a.Form.Media()
}
func (a *AdminForm[T1, T2]) Prefix() string {
	return a.Form.Prefix()
}
func (a *AdminForm[T1, T2]) SetPrefix(prefix string) {
	a.Form.SetPrefix(prefix)
}
func (a *AdminForm[T1, T2]) SetInitial(initial map[string]interface{}) {
	a.Form.SetInitial(initial)
}
func (a *AdminForm[T1, T2]) SetValidators(validators ...func(forms.Form, map[string]interface{}) []error) {
	a.Form.SetValidators(validators...)
}
func (a *AdminForm[T1, T2]) Ordering(o []string) {
	a.Form.Ordering(o)
}
func (a *AdminForm[T1, T2]) FieldOrder() []string {
	return a.Form.FieldOrder()
}
func (a *AdminForm[T1, T2]) Field(name string) (fields.Field, bool) {
	return a.Form.Field(name)
}
func (a *AdminForm[T1, T2]) Widget(name string) (widgets.Widget, bool) {
	return a.Form.Widget(name)
}
func (a *AdminForm[T1, T2]) Fields() []fields.Field {
	return a.Form.Fields()
}
func (a *AdminForm[T1, T2]) Widgets() []widgets.Widget {
	return a.Form.Widgets()
}
func (a *AdminForm[T1, T2]) AddField(name string, field fields.Field) {
	a.Form.AddField(name, field)
}
func (a *AdminForm[T1, T2]) DeleteField(name string) bool {
	return a.Form.DeleteField(name)
}
func (a *AdminForm[T1, T2]) AddWidget(name string, widget widgets.Widget) {
	a.Form.AddWidget(name, widget)
}

func (a *AdminForm[T1, T2]) BoundForm() forms.BoundForm {
	var form = a.Form.BoundForm()
	return NewPanelBoundForm(
		a.Context(),
		a.Request,
		a.Form.Instance(),
		a.Form,
		form,
		a.Panels,
	)
}
func (a *AdminForm[T1, T2]) BoundFields() *orderedmap.OrderedMap[string, forms.BoundField] {
	return a.Form.BoundFields()
}
func (a *AdminForm[T1, T2]) BoundErrors() *orderedmap.OrderedMap[string, []error] {
	return a.Form.BoundErrors()
}
func (a *AdminForm[T1, T2]) ErrorList() []error {
	return a.Form.ErrorList()
}
func (a *AdminForm[T1, T2]) WithData(data url.Values, files map[string][]filesystem.FileHeader, r *http.Request) forms.Form {
	a.Request = r
	return a.Form.WithData(data, files, r)
}
func (a *AdminForm[T1, T2]) InitialData() map[string]interface{} {
	return a.Form.InitialData()
}
func (a *AdminForm[T1, T2]) CleanedData() map[string]interface{} {
	return a.Form.CleanedData()
}
func (a *AdminForm[T1, T2]) HasChanged() bool {
	var (
		fields  = make([]string, 0)
		fieldsM = make(map[string]struct{})
		initial = a.InitialData()
		cleaned = a.CleanedData()
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
		if f.ReadOnly() {
			continue
		}

		if f.HasChanged(initial[fieldName], cleaned[fieldName]) {
			return true
		}
	}

	return false

}
func (a *AdminForm[T1, T2]) PrefixName(name string) (prefixedName string) {
	return a.Form.PrefixName(name)
}
func (a *AdminForm[T1, T2]) FieldMap() *orderedmap.OrderedMap[string, forms.Field] {
	return a.Form.FieldMap()
}

func (f *AdminForm[T1, T2]) Validators() []func(forms.Form, map[string]interface{}) []error {
	return f.Form.Validators()
}

func (f *AdminForm[T1, T2]) CallbackOnValid() []func(forms.Form) {
	return f.Form.CallbackOnValid()
}

func (f *AdminForm[T1, T2]) CallbackOnInvalid() []func(forms.Form) {
	return f.Form.CallbackOnInvalid()
}

func (f *AdminForm[T1, T2]) CallbackOnFinalize() []func(forms.Form) {
	return f.Form.CallbackOnFinalize()
}

func (f *AdminForm[T1, T2]) BindCleanedData(invalid, defaults, cleaned map[string]interface{}) {
	f.Form.BindCleanedData(invalid, defaults, cleaned)
}

func (f *AdminForm[T1, T2]) CleanedDataUnsafe() map[string]interface{} {
	return f.Form.CleanedDataUnsafe()
}

func (f *AdminForm[T1, T2]) Data() (url.Values, map[string][]filesystem.FileHeader) {
	return f.Form.Data()
}
func (f *AdminForm[T1, T2]) WasCleaned() bool {
	return f.Form.WasCleaned()
}
func (a *AdminForm[T1, T2]) AddFormError(errorList ...error) {
	a.Form.AddFormError(errorList...)
}
func (a *AdminForm[T1, T2]) AddError(name string, errorList ...error) {
	a.Form.AddError(name, errorList...)
}
func (a *AdminForm[T1, T2]) OnValid(f ...func(forms.Form)) {
	a.Form.OnValid(f...)
}
func (a *AdminForm[T1, T2]) OnInvalid(f ...func(forms.Form)) {
	a.Form.OnInvalid(f...)
}
func (a *AdminForm[T1, T2]) OnFinalize(f ...func(forms.Form)) {
	a.Form.OnFinalize(f...)
}
func (a *AdminForm[T1, T2]) IsValid() bool {
	if validDef, ok := any(a.Form).(forms.IsValidDefiner); ok {
		return validDef.IsValid()
	}
	return true
}

func (a *AdminForm[T1, T2]) SetOnLoad(fn func(model T2, initialData map[string]interface{})) {
	a.Form.SetOnLoad(fn)
}

func (a *AdminForm[T1, T2]) Load() {
	a.Form.Load()

	var fields = a.FieldMap()
	if len(a.Panels) == 0 {
		a.Panels = make([]Panel, 0, fields.Len())
		for _, key := range fields.Keys() {
			a.Panels = append(
				a.Panels,
				FieldPanel(key),
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

	for _, key := range fields.Keys() {
		if _, ok := panelFields[key]; !ok {
			a.Form.DeleteField(key)
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

func (a *AdminForm[T1, T2]) Save() (map[string]interface{}, error) {
	var data, err = a.Form.Save()
	if err != nil {
		return data, err
	}
	return data, nil
}
func (a *AdminForm[T1, T2]) SetFields(fields ...string) {
	a.Form.SetFields(fields...)
}
func (a *AdminForm[T1, T2]) SetExclude(exclude ...string) {
	a.Form.SetExclude(exclude...)
}
func (a *AdminForm[T1, T2]) Instance() attrs.Definer {
	return a.Form.Instance()
}
func (a *AdminForm[T1, T2]) SetInstance(model T2) {
	a.Form.SetInstance(model)
}
