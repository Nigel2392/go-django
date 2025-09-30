package forms

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strings"

	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/elliotchance/orderedmap/v2"
)

type BaseForm struct {
	FormPrefix      string
	fieldOrder      []string
	FormFields      *orderedmap.OrderedMap[string, Field]
	FormWidgets     *orderedmap.OrderedMap[string, Widget]
	Errors          *orderedmap.OrderedMap[string, []error]
	ErrorList_      []error
	Raw             url.Values
	Initial         map[string]interface{}
	InvalidDefaults map[string]interface{}
	Files           map[string][]filesystem.FileHeader
	Cleaned         map[string]interface{}
	Defaults        map[string]interface{}
	FormContext     context.Context

	FormValidators  []func(Form, map[string]interface{}) []error
	OnValidFuncs    []func(Form)
	OnInvalidFuncs  []func(Form)
	OnFinalizeFuncs []func(Form)
}

func NewBaseForm(ctx context.Context, opts ...func(Form)) *BaseForm {
	var f = &BaseForm{
		FormFields:      orderedmap.NewOrderedMap[string, Field](),
		FormWidgets:     orderedmap.NewOrderedMap[string, Widget](),
		Initial:         make(map[string]interface{}),
		InvalidDefaults: make(map[string]interface{}),
		FormContext:     ctx,
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

func (f *BaseForm) setup() {
	if f.FormFields == nil {
		f.FormFields = orderedmap.NewOrderedMap[string, Field]()
	}

	if f.FormWidgets == nil {
		f.FormWidgets = orderedmap.NewOrderedMap[string, Widget]()
	}

	if f.Errors == nil {
		f.Errors = orderedmap.NewOrderedMap[string, []error]()
	}

	if f.ErrorList_ == nil {
		f.ErrorList_ = make([]error, 0)
	}

	if f.Initial == nil {
		f.Initial = make(map[string]interface{})
	}

	if f.InvalidDefaults == nil {
		f.InvalidDefaults = make(map[string]interface{})
	}
}

func (f *BaseForm) Context() context.Context {
	return f.FormContext
}

func (f *BaseForm) WithContext(ctx context.Context) {
	f.FormContext = ctx
}

func (f *BaseForm) FormValue(name string) interface{} {

	if f.Cleaned != nil {
		if v, ok := f.Cleaned[name]; ok {
			return v
		}
	}

	if f.InvalidDefaults != nil {
		if v, ok := f.InvalidDefaults[name]; ok {
			return v
		}
	}

	if f.Defaults != nil {
		if v, ok := f.Defaults[name]; ok {
			return v
		}
	}

	return nil
}

func (f *BaseForm) ErrorList() []error {
	return f.ErrorList_
}

func (f *BaseForm) BoundErrors() *orderedmap.OrderedMap[string, []error] {
	var errs = f.Errors
	if len(f.ErrorList_) > 0 {
		if f.Errors == nil {
			f.Errors = orderedmap.NewOrderedMap[string, []error]()
		}
		errs = f.Errors.Copy()
		errs.Set("__all__", f.ErrorList_)
	}
	if errs == nil || errs.Len() == 0 {
		return nil
	}
	return errs
}

func (f *BaseForm) BoundForm() BoundForm {
	return NewBoundForm(f)
}

func (f *BaseForm) EditContext(key string, context ctx.Context) {
	context.Set(key, f.BoundForm())
}

func (f *BaseForm) AddField(name string, field Field) {
	field.SetName(name)
	f.FormFields.Set(name, field)
}

func (f *BaseForm) DeleteField(name string) bool {
	return f.FormFields.Delete(name)
}

func (f *BaseForm) AddWidget(name string, widget Widget) {
	f.FormWidgets.Set(name, widget)
}

func (f *BaseForm) FieldMap() *orderedmap.OrderedMap[string, Field] {
	if f.FormFields == nil {
		f.FormFields = orderedmap.NewOrderedMap[string, Field]()
	}
	return f.FormFields
}

func (f *BaseForm) BoundFields() *orderedmap.OrderedMap[string, BoundField] {
	f.setup()

	var ret = orderedmap.NewOrderedMap[string, BoundField]()
	for head := f.FormFields.Front(); head != nil; head = head.Next() {
		var (
			k          = head.Key
			v          = head.Value
			widget, ok = f.FormWidgets.Get(k)
			errors, _  = f.Errors.Get(k)
		)
		if !ok {
			widget = v.Widget()
		}

		var value = f.FormValue(k)
		var rV = reflect.ValueOf(value)
		if (rV.Kind() == reflect.Interface || rV.Kind() == reflect.Ptr) && rV.IsNil() {
			value = nil
		}

		value = v.ValueToForm(
			value,
		)

		ret.Set(k, NewBoundFormField(
			f.FormContext,
			widget,
			v,
			f.PrefixName(k),
			value,
			errors,
			true,
		))
	}
	return ret
}

func (f *BaseForm) OnValid(funcs ...func(Form)) {
	f.OnValidFuncs = append(f.OnValidFuncs, funcs...)
}

func (f *BaseForm) OnInvalid(funcs ...func(Form)) {
	f.OnInvalidFuncs = append(f.OnInvalidFuncs, funcs...)
}

func (f *BaseForm) OnFinalize(funcs ...func(Form)) {
	f.OnFinalizeFuncs = append(f.OnFinalizeFuncs, funcs...)
}

func (f *BaseForm) AsP() template.HTML {
	var bound = f.BoundFields()
	var html = make([]string, 0, bound.Len())
	for head := bound.Front(); head != nil; head = head.Next() {
		var (
			label = head.Value.Label()
			field = head.Value.Field()
		)
		html = append(html, fmt.Sprintf("<p>%s %s</p>", label, field))
	}
	return template.HTML(strings.Join(html, ""))
}

func (f *BaseForm) AsUL() template.HTML {
	var bound = f.BoundFields()
	var html = make([]string, 0, bound.Len()*2)
	for head := bound.Front(); head != nil; head = head.Next() {
		var (
			label = head.Value.Label()
			field = head.Value.Field()
		)
		html = append(html, fmt.Sprintf("\t<li>%s</li>\n", label))
		html = append(html, fmt.Sprintf("\t<li>%s</li>\n", field))
	}
	return template.HTML(fmt.Sprintf("<ul>\n%s\n</ul>", strings.Join(html, "")))
}

func (f *BaseForm) Media() media.Media {
	var m media.Media = media.NewMedia()
	if f.FormWidgets == nil {
		f.FormWidgets = orderedmap.NewOrderedMap[string, Widget]()
	}

	for head := f.FormFields.Front(); head != nil; head = head.Next() {

		var widget, ok = f.FormWidgets.Get(head.Key)
		if !ok {
			widget = head.Value.Widget()
		}

		var wm = widget.Media()
		if wm != nil {
			m = m.Merge(wm)
		}
	}

	return m
}

func (f *BaseForm) Fields() []Field {
	var inputs = make([]Field, 0, f.FormFields.Len())
	for head := f.FormFields.Front(); head != nil; head = head.Next() {
		inputs = append(inputs, head.Value)
	}
	return inputs
}

func (f *BaseForm) Widgets() []Widget {
	var widgets = make([]Widget, 0, f.FormWidgets.Len())
	for head := f.FormWidgets.Front(); head != nil; head = head.Next() {
		widgets = append(widgets, head.Value)
	}
	return widgets
}

func (f *BaseForm) Field(name string) (Field, bool) {
	return f.FormFields.Get(name)
}

func (f *BaseForm) Widget(name string) (Widget, bool) {
	var widget, ok = f.FormWidgets.Get(name)
	var field Field
	if !ok {
		field, ok = f.FormFields.Get(name)
		if ok {
			widget = field.Widget()
		}
	}
	return widget, ok
}

func (f *BaseForm) InitialData() map[string]interface{} {
	return f.Initial
}

func (f *BaseForm) CleanedData() map[string]interface{} {
	if f.Errors != nil {
		except.Assert(
			f.Errors.Len() == 0, 500,
			"You cannot access cleaned data if the form is invalid: there are %d errors for fields %v",
			f.Errors.Len(), f.Errors.Keys(),
		)
	}
	if len(f.ErrorList_) > 0 {
		except.Assert(len(f.ErrorList_) == 0, 500, "You cannot access cleaned data if the form has errors.")
	}
	except.Assert(f.Cleaned != nil, 500, "You must call IsValid() before accessing cleaned data.")
	return f.Cleaned
}

func (f *BaseForm) SetInitial(initial map[string]interface{}) {
	f.Initial = initial
	f.Defaults = initial
}

func (f *BaseForm) SetPrefix(prefix string) {
	f.FormPrefix = prefix
}

func (f *BaseForm) Prefix() string {
	return f.FormPrefix
}

func (f *BaseForm) PrefixName(name string) string {
	var prefix = f.Prefix()
	if prefix == "" {
		return name
	}
	return fmt.Sprintf("%s-%s", prefix, name)
}

func (f *BaseForm) WithData(data url.Values, files map[string][]filesystem.FileHeader, r *http.Request) {
	f.Reset()
	f.Raw = data
	f.Files = files
	f.setup()
}

func (f *BaseForm) Reset() {
	f.Raw = nil
	f.Initial = nil
	f.Errors = nil
	f.ErrorList_ = nil
	f.InvalidDefaults = nil
	f.Files = nil
	f.Cleaned = nil
	f.Defaults = nil
}

func (f *BaseForm) Ordering(order []string) {
	f.fieldOrder = order
}

func (f *BaseForm) FieldOrder() []string {
	return f.fieldOrder
}

func (f *BaseForm) SetValidators(validators ...func(Form, map[string]interface{}) []error) {
	if f.FormValidators == nil {
		f.FormValidators = make([]func(Form, map[string]interface{}) []error, 0)
	}
	f.FormValidators = append(f.FormValidators, validators...)
}

func (f *BaseForm) addErrors(errorList ...error) {
	if f.ErrorList_ == nil {
		f.ErrorList_ = make([]error, 0, len(errorList))
	}

	for _, err := range errorList {
		switch e := err.(type) {
		case interface{ Unwrap() []error }:
			f.addErrors(e.Unwrap()...)
		case errs.ValidationError[string]:
			f.AddError(e.Name, e.Err)
		default:
			f.ErrorList_ = append(f.ErrorList_, err)
		}
	}
}

func (f *BaseForm) AddFormError(errorList ...error) {
	if f.ErrorList_ == nil {
		f.ErrorList_ = make([]error, 0)
	}

	var newErrs = slices.Clone(errorList)
	f.addErrors(newErrs...)
}

func (f *BaseForm) AddError(name string, errorList ...error) {
	if f.Errors == nil {
		f.Errors = orderedmap.NewOrderedMap[string, []error]()
	}

	var head, ok = f.Errors.Get(name)
	if !ok {
		head = make([]error, 0)
	}

	var newErrs = slices.Clone(errorList)
loop:
	for i, err := range newErrs {
		switch e := err.(type) {
		case errs.DjangoError:
			continue loop
		default:
			newErrs[i] = errs.NewValidationError(name, e)
		}
	}

	head = append(head, newErrs...)

	if len(head) > 0 {
		var _, ok = f.Field(name)
		if ok {
			f.Errors.Set(name, head)
		} else {
			f.ErrorList_ = append(f.ErrorList_, head...)
		}
	}
}

func (f *BaseForm) HasChanged() bool {
	var changed bool = false

	for head := f.FormFields.Front(); head != nil; head = head.Next() {
		var (
			k     = head.Key
			field = head.Value
		)
		var v, ok = f.Widget(k)
		if !ok {
			v = field.Widget()
		}

		if _, ok := f.Initial[k]; !ok {
			var omitted = v.ValueOmittedFromData(f.FormContext, f.Raw, f.Files, f.PrefixName(k))
			if !omitted {
				return true
			}
		}

		if f.Initial[k] != f.Cleaned[k] {
			return true
		}
	}

	return changed
}

func (f *BaseForm) Save() (map[string]interface{}, error) {
	if f.Errors != nil && f.Errors.Len() > 0 {
		return nil, errs.Error("the form cannot be saved because it has errors")
	}

	if f.Cleaned == nil {
		except.Assert(f.Cleaned != nil, 500, "You must call IsValid() before calling Save().")
	}

	var err error
	var data = f.CleanedData()
	for head := f.FormFields.Front(); head != nil; head = head.Next() {
		var (
			k     = head.Key
			field = head.Value
		)

		if field.ReadOnly() {
			data[k] = f.Initial[k]
			continue
		}

		var value, ok = data[k]
		if !ok {
			continue
		}

		// Check if the field is saveable and call Save() on it.
		// This might be used to save a relation to the database, among other things.
		if field, saveable := field.(fields.SaveableField); saveable {
			value, err = field.Save(value)
			if err != nil {
				return nil, err
			}

			data[k] = value
		}
	}

	return data, nil
}

func (f *BaseForm) Validators() []func(f Form, cleanedData map[string]interface{}) []error {
	return f.FormValidators
}

func (f *BaseForm) CallbackOnValid() []func(Form) {
	return f.OnValidFuncs
}

func (f *BaseForm) CallbackOnInvalid() []func(Form) {
	return f.OnInvalidFuncs
}

func (f *BaseForm) CallbackOnFinalize() []func(Form) {
	return f.OnFinalizeFuncs
}

func (f *BaseForm) BindCleanedData(invalid, defaults, cleaned map[string]interface{}) {
	f.InvalidDefaults = invalid
	f.Defaults = defaults
	f.Cleaned = cleaned
}

func (f *BaseForm) CleanedDataUnsafe() map[string]interface{} {
	return f.Cleaned
}

func (f *BaseForm) Data() (url.Values, map[string][]filesystem.FileHeader) {
	return f.Raw, f.Files
}

func (f *BaseForm) WasCleaned() bool {
	return len(f.Cleaned) > 0 || (f.Errors != nil && f.Errors.Len() > 0) || len(f.ErrorList_) > 0
}
