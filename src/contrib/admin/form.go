package admin

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"slices"

	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/formsets"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/Nigel2392/go-django/src/models"
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
	FormOrder() FORM_ORDERING
}

type AdminForm[T1 modelforms.ModelForm[T2], T2 attrs.Definer] struct {
	Form    T1
	Panels  []Panel
	Request *http.Request

	forms   FormSetMap
	formset formsets.ListFormSet[formsets.BaseFormSetForm]
}

func NewAdminForm[T1 modelforms.ModelForm[T2], T2 attrs.Definer](r *http.Request, form T1, panels ...Panel) *AdminForm[T1, T2] {
	return &AdminForm[T1, T2]{
		Form:    form,
		Panels:  panels,
		Request: r,
	}
}

func (a *AdminForm[T1, T2]) Unwrap() []formsets.BaseFormSetForm {
	var fs = a.FormSet()
	if fs == nil {
		return []formsets.BaseFormSetForm{a}
	}

	return []formsets.BaseFormSetForm{a, fs}
}

func (a *AdminForm[T1, T2]) EditContext(key string, context ctx.Context) {
	context.Set(key, a.BoundForm())
}

func (a *AdminForm[T1, T2]) Context() context.Context {
	return a.Form.Context()
}

func (a *AdminForm[T1, T2]) WithContext(ctx context.Context) {
	a.Form.WithContext(ctx)
	var formset = a.FormSet()
	if formset != nil {
		formset.WithContext(ctx)
	}
}

func (a *AdminForm[T1, T2]) Media() media.Media {
	return a.Form.Media()
}
func (a *AdminForm[T1, T2]) Prefix() string {
	return a.Form.Prefix()
}
func (a *AdminForm[T1, T2]) SetPrefix(prefix string) {
	a.Form.SetPrefix(prefix)

	if a.FormSet() != nil {
		a.FormSet().SetPrefix(prefix)
	}
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

func (a *AdminForm[T1, T2]) UnpackErrors(bound forms.BoundForm, boundErrors *orderedmap.OrderedMap[string, []error]) []forms.FieldError {
	var errors = make([]forms.FieldError, 0)
	errors = append(errors, forms.UnpackErrors(bound, a.Form, boundErrors, func(s string) (forms.Field, bool) {
		return a.Form.Field(s)
	})...)

	if a.formset != nil {
		if mgmt := a.formset.ManagementForm(); mgmt != nil {
			errors = append(errors, forms.UnpackErrors(bound, mgmt, mgmt.BoundErrors(), func(s string) (forms.Field, bool) {
				return mgmt.Field(s)
			})...)
		}
	}
	return errors
}

func (a *AdminForm[T1, T2]) BoundForm() forms.BoundForm {
	var form = forms.NewBoundForm(a.Context(), a, nil)

	return NewPanelBoundForm(
		a.Context(), a.Request, a.Form.Instance(),
		a, form, a.Panels, a.forms,
	)
}

func (a *AdminForm[T1, T2]) BoundFields() *orderedmap.OrderedMap[string, forms.BoundField] {
	return a.Form.BoundFields()
}
func (a *AdminForm[T1, T2]) BoundErrors() *orderedmap.OrderedMap[string, []error] {
	return a.Form.BoundErrors()
}
func (a *AdminForm[T1, T2]) ErrorList() []error {
	var errList = a.Form.ErrorList()
	if a.formset != nil {
		var mgmt = a.formset.ManagementForm()
		if mgmt != nil {
			errList = append(errList, mgmt.ErrorList()...)
		}
		errList = append(errList, a.formset.ErrorList()...)
	}
	return errList
}
func (a *AdminForm[T1, T2]) WithData(data url.Values, files map[string][]filesystem.FileHeader, r *http.Request) {
	a.Request = r
	a.Form.WithData(data, files, r)
	if a.formset != nil {
		a.formset.WithData(data, files, r)
	}
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
		var f, ok = a.Form.Field(fieldName)
		if !ok {
			continue
		}

		if f.ReadOnly() {
			continue
		}

		if f.HasChanged(initial[fieldName], cleaned[fieldName]) {
			return true
		}
	}

	if a.formset != nil && a.formset.HasChanged() {
		return true
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
	if validDef, ok := any(a.Form).(forms.IsValidDefiner); ok && !validDef.IsValid() {
		return false
	}
	var fs = a.FormSet()
	if fs != nil {
		if !forms.IsValid(a.Context(), fs) {
			return false
		}
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

	if a.FormSet() != nil {
		a.FormSet().Load()
	}
}

func (a *AdminForm[T1, T2]) FormSet() formsets.ListFormSet[formsets.BaseFormSetForm] {
	if a.formset != nil {
		return a.formset
	}

	var formsList = make([]formsets.BaseFormSetForm, 0)
	if unwrapper, ok := any(a.Form).(forms.FormWrapper[forms.Form]); ok {
		var f = unwrapper.Unwrap()
		var nl = make([]formsets.BaseFormSetForm, len(f))
		for i, v := range f {
			nl[i] = v.(formsets.BaseFormSetForm)
		}
		formsList = append(formsList, nl...)
	}
	if unwrapper, ok := any(a.Form).(forms.FormWrapper[any]); ok {
		var f = unwrapper.Unwrap()
		var nl = make([]formsets.BaseFormSetForm, len(f))
		for i, v := range f {
			nl[i] = v.(formsets.BaseFormSetForm)
		}
		formsList = append(formsList, nl...)
	}
	if unwrapper, ok := any(a.Form).(forms.FormWrapper[T1]); ok {
		var f = unwrapper.Unwrap()
		var nl = make([]formsets.BaseFormSetForm, len(f))
		for i, v := range f {
			nl[i] = any(v).(formsets.BaseFormSetForm)
		}
		formsList = append(formsList, nl...)
	}

	// var errs = make([]error, 0)
	var formSets FormSetMap = make(map[string]FormSetObject)
	for i, panel := range a.Panels {
		if unwrapper, ok := panel.(FormPanel); ok {
			fMap, formList, err := unwrapper.Forms(a.Request, a.Context(), a.Instance())
			if err != nil {
				logger.Errorf("could not get forms from panel %T: %v", unwrapper, err)
				continue
			}

			formSets[panelPathPart(panel, i)] = fMap
			formsList = append(formsList, formList...)
		}
	}

	slices.SortStableFunc(formsList, func(i, j formsets.BaseFormSetForm) int {
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

	a.forms = formSets
	a.formset = formsets.NewBaseFormSet(
		a.Context(),
		formsets.FormsetOptions[formsets.BaseFormSetForm]{
			MinNum:     len(formsList),
			MaxNum:     len(formsList),
			Extra:      0,
			CanDelete:  false,
			CanAdd:     false,
			CanOrder:   false,
			SkipPrefix: true,
		},
	)
	a.formset.SetForms(formsList)
	return a.formset

	//if len(errs) > 0 {
	//	return formsList //, errors.Join(errs...)
	//}
	//
	//return formsList //, nil
}

func (a *AdminForm[T1, T2]) Save() (map[string]interface{}, error) {
	var data, err = a.Form.Save()
	if err != nil {
		return data, err
	}

	return data, nil
}

func (a *AdminForm[T1, T2]) SaveForms(formList ...forms.Form) (err error) {
	if len(formList) == 0 && len(a.forms) == 0 {
		a.FormSet()
	}

	var flist = make([]formsets.BaseFormSetForm, 0, len(formList))
	if len(formList) == 0 {
		flist, err = a.FormSet().Forms()
		if err != nil {
			return err
		}
	} else {
		for _, f := range formList {
			flist = append(flist, f)
		}
	}

	for _, form := range flist {
		var rV = reflect.ValueOf(form)
		var saveMethod = rV.MethodByName("Save")
		if !saveMethod.IsValid() {
			logger.Warnf("could not save form, no Save method found on %T", form)
			continue
		}

		if saveMethod.Type().NumIn() != 0 {
			logger.Warnf("could not save form, Save method on %T has %d inputs, expected 0", form, saveMethod.Type().NumIn())
			continue
		}

		var cleaned = form.CleanedData()
		var deleted, _ = cleaned["__DELETED__"].(string)
		if deleted == "true" {
			var instanceMethod = rV.MethodByName("Instance")
			if !instanceMethod.IsValid() {
				// ? maybe do something here, not sure..
				logger.Warnf("could not delete form, no Instance method found on %T", form)
				continue
			}

			if instanceMethod.Type().NumIn() != 0 {
				// ? maybe do something here, not sure..
				logger.Warnf("could not delete form, Instance method on %T has %d inputs, expected 0", form, instanceMethod.Type().NumIn())
				continue
			}

			if instanceMethod.Type().NumOut() < 1 {
				// ? maybe do something here, not sure..
				logger.Warnf("could not delete form, Instance method on %T has %d outputs, expected at least 1", form, instanceMethod.Type().NumOut())
				continue
			}

			if !instanceMethod.Type().Out(0).ConvertibleTo(reflect.TypeOf((*attrs.Definer)(nil)).Elem()) {
				// ? maybe do something here, not sure..
				logger.Warnf("could not delete form, Instance method on %T does not return an attrs.Definer, got %v", form, instanceMethod.Type().Out(0))
				continue
			}

			var vals = instanceMethod.Call([]reflect.Value{})
			var instanceVal = vals[0].Interface()
			if instanceVal == nil {
				// ? maybe do something here, not sure..
				logger.Warnf("could not delete form, Instance method on %T returned nil", form)
				continue
			}

			logger.Debugf("AdminForm: deleting form %T, instance: %v", form, instanceVal)
			deleted, err := models.DeleteModel(a.Context(), instanceVal.(attrs.Definer))
			if err != nil {
				logger.Errorf("could not delete model %T: %v", instanceVal, err)
				a.Form.AddFormError(err)
				continue
			}
			if !deleted {
				logger.Warnf("could not delete model %T: unknown error", instanceVal)
				a.Form.AddFormError(errors.New("could not delete model"))
			}
			continue
		}

		var results = saveMethod.Call([]reflect.Value{})
		if len(results) > 0 {
			var last = results[len(results)-1]
			if !last.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) || last.IsNil() {
				continue
			}

			err = last.Interface().(error)
		}
		if err != nil {
			a.Form.AddFormError(err)
		}
	}
	return nil
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
