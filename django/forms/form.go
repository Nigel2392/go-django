package forms

import (
	"fmt"
	"html/template"
	"io"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/django/core/ctx"
	"github.com/Nigel2392/django/core/errs"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/django/forms/media"
	"github.com/Nigel2392/django/forms/widgets"
	"github.com/elliotchance/orderedmap/v2"
)

func WithRequestData(method string, r *http.Request) func(Form) {
	if r.Method != method {
		return func(f Form) {
			var (
				data  = make(url.Values)
				files = make(map[string][]io.ReadCloser)
			)
			f.WithData(data, files, r)
		}
	}

	return func(f Form) {
		r.ParseForm()

		var data = make(url.Values)
		maps.Copy(data, r.Form)
		var files = make(map[string][]io.ReadCloser)
		if r.MultipartForm != nil && r.MultipartForm.File != nil {
			for k, v := range r.MultipartForm.File {
				var files_ = make([]io.ReadCloser, 0, len(v))
				for _, file := range v {
					var file, err = file.Open()
					assert.ErrNil(err)
					files_ = append(files_, file)
				}
				files[k] = files_
			}
		}

		f.WithData(data, files, r)
	}
}

func WithData(data url.Values, files map[string][]io.ReadCloser, r *http.Request) func(Form) {
	if files == nil {
		files = make(map[string][]io.ReadCloser)
	}

	return func(f Form) {
		f.WithData(data, files, r)
	}
}

func WithFields(fields ...fields.Field) func(Form) {
	return func(f Form) {
		for _, field := range fields {
			f.AddField(field.Name(), field)
		}
	}
}

func WithPrefix(prefix string) func(Form) {
	return func(f Form) {
		f.SetPrefix(prefix)
	}
}

func WithInitial(initial map[string]interface{}) func(Form) {
	return func(f Form) {
		f.SetInitial(initial)
	}
}

func OnValid(funcs ...func(Form)) func(Form) {
	return func(f Form) {
		f.OnValid(funcs...)
	}
}

func OnInvalid(funcs ...func(Form)) func(Form) {
	return func(f Form) {
		f.OnInvalid(funcs...)
	}
}

func OnFinalize(funcs ...func(Form)) func(Form) {
	return func(f Form) {
		f.OnFinalize(funcs...)
	}
}

func Initialize[T Form](f T, initfuncs ...func(Form)) T {

	for _, initfunc := range initfuncs {
		initfunc(f)
	}

	return f
}

func (f *BaseForm) setup() {
	if f.FormFields == nil {
		f.FormFields = orderedmap.NewOrderedMap[string, fields.Field]()
	}

	if f.FormWidgets == nil {
		f.FormWidgets = orderedmap.NewOrderedMap[string, widgets.Widget]()
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

type BaseForm struct {
	FormPrefix      string
	fieldOrder      []string
	FormFields      *orderedmap.OrderedMap[string, fields.Field]
	FormWidgets     *orderedmap.OrderedMap[string, widgets.Widget]
	Errors          *orderedmap.OrderedMap[string, []error]
	ErrorList_      []error
	Raw             url.Values
	Initial         map[string]interface{}
	InvalidDefaults map[string]interface{}
	Files           map[string][]io.ReadCloser
	Cleaned         map[string]interface{}

	Validators      []func(Form) []error
	OnValidFuncs    []func(Form)
	OnInvalidFuncs  []func(Form)
	OnFinalizeFuncs []func(Form)
}

func NewBaseForm(opts ...func(Form)) *BaseForm {
	var f = &BaseForm{
		FormFields:      orderedmap.NewOrderedMap[string, fields.Field](),
		FormWidgets:     orderedmap.NewOrderedMap[string, widgets.Widget](),
		Initial:         make(map[string]interface{}),
		InvalidDefaults: make(map[string]interface{}),
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

func (f *BaseForm) DefaultValue(name string) interface{} {

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

	if f.Initial != nil {
		if v, ok := f.Initial[name]; ok {
			return v
		}
	}

	return nil
}

func (f *BaseForm) ErrorList() []error {
	return f.ErrorList_
}

func (f *BaseForm) BoundErrors() *orderedmap.OrderedMap[string, []error] {
	if f.Errors == nil && (len(f.Raw) > 0 || len(f.Files) > 0) {
		f.FullClean()
	}

	var errs = f.Errors
	if len(f.ErrorList_) > 0 {
		if f.Errors == nil {
			f.Errors = orderedmap.NewOrderedMap[string, []error]()
		}
		errs = f.Errors.Copy()
		errs.Set("__all__", f.ErrorList_)
	}
	if errs.Len() == 0 {
		return nil
	}
	return errs
}

type BoundForm interface {
	AsP() template.HTML
	AsUL() template.HTML
	Media() media.Media
	Fields() []BoundField
	ErrorList() []error
	Errors() *orderedmap.OrderedMap[string, []error]
}

type _BoundForm struct {
	Form       Form
	Fields_    []BoundField
	Errors_    *orderedmap.OrderedMap[string, []error]
	ErrorList_ []error
}

func (f *_BoundForm) AsP() template.HTML {
	return f.Form.AsP()
}

func (f *_BoundForm) AsUL() template.HTML {
	return f.Form.AsUL()
}

func (f *_BoundForm) Fields() []BoundField {
	return f.Fields_
}

func (f *_BoundForm) Errors() *orderedmap.OrderedMap[string, []error] {
	return f.Errors_
}

func (f *_BoundForm) ErrorList() []error {
	return f.ErrorList_
}

func (f *_BoundForm) Media() media.Media {
	return f.Form.Media()
}

func (f *BaseForm) BoundForm() BoundForm {
	var (
		fields      = f.BoundFields()
		errors      = f.BoundErrors()
		boundFields = make([]BoundField, 0, fields.Len())
	)
	//for head := fields.Front(); head != nil; head = head.Next() {
	//	boundFields = append(boundFields, head.Value)
	//}
	if f.fieldOrder != nil {
		var had = make(map[string]struct{})
		for _, k := range f.fieldOrder {
			if v, ok := fields.Get(k); ok {
				boundFields = append(boundFields, v)
				had[k] = struct{}{}
			}
		}
		if fields.Len() > len(had) {
			for head := fields.Front(); head != nil; head = head.Next() {
				var (
					k = head.Key
					v = head.Value
				)
				if _, ok := had[k]; !ok {
					boundFields = append(boundFields, v)
				}
			}
		}
	} else {
		for head := fields.Front(); head != nil; head = head.Next() {
			boundFields = append(boundFields, head.Value)
		}
	}

	return &_BoundForm{
		Form:       f,
		Fields_:    boundFields,
		Errors_:    errors,
		ErrorList_: f.ErrorList_,
	}
}

func (f *BaseForm) EditContext(key string, context ctx.Context) {
	context.Set(key, f.BoundForm())
}

func (f *BaseForm) AddField(name string, field fields.Field) {
	field.SetName(name)
	f.FormFields.Set(name, field)
}

func (f *BaseForm) AddWidget(name string, widget widgets.Widget) {
	f.FormWidgets.Set(name, widget)
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
			value      interface{}
		)
		if !ok {
			widget = v.Widget()
		}

		value = v.ValueToForm(
			f.DefaultValue(k),
		)

		ret.Set(k, NewBoundFormWidget(
			widget,
			v,
			k,
			value,
			errors,
		))
	}
	return ret
}

func (f *BaseForm) Close() (err error) {
	var errs = make([]error, 0)
	for _, files := range f.Files {
		for _, file := range files {
			if err = file.Close(); err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(errs) > 0 {
		var errStr = make([]string, 0, len(errs))
		for _, err := range errs {
			errStr = append(errStr, err.Error())
		}
		return fmt.Errorf(
			"error(s) closing files: %s", strings.Join(errStr, ", "),
		)
	}

	return nil
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
	var media media.Media = media.NewMedia()
	if f.FormWidgets == nil {
		f.FormWidgets = orderedmap.NewOrderedMap[string, widgets.Widget]()
	}

	for head := f.FormFields.Front(); head != nil; head = head.Next() {

		var widget, ok = f.FormWidgets.Get(head.Key)
		if !ok {
			widget = head.Value.Widget()
		}

		var m = widget.Media()
		if m != nil {
			media = media.Merge(m)
		}
	}

	return media
}

func (f *BaseForm) Fields() []fields.Field {
	var inputs = make([]fields.Field, 0, f.FormFields.Len())
	for head := f.FormFields.Front(); head != nil; head = head.Next() {
		inputs = append(inputs, head.Value)
	}
	return inputs
}

func (f *BaseForm) Widgets() []widgets.Widget {
	var widgets = make([]widgets.Widget, 0, f.FormWidgets.Len())
	for head := f.FormWidgets.Front(); head != nil; head = head.Next() {
		widgets = append(widgets, head.Value)
	}
	return widgets
}

func (f *BaseForm) Field(name string) fields.Field {
	var ret, ok = f.FormFields.Get(name)
	assert.True(ok, "The input %s does not exist.", name)
	return ret
}

func (f *BaseForm) Widget(name string) widgets.Widget {
	var ret, ok = f.FormWidgets.Get(name)
	assert.True(ok, "The widget %s does not exist.", name)
	return ret
}

func (f *BaseForm) CleanedData() map[string]interface{} {
	if f.Errors != nil {
		assert.True(f.Errors.Len() == 0, "You cannot access cleaned data if the form is invalid.")
	}
	assert.False(f.Cleaned == nil, "You must call IsValid() before accessing cleaned data.")
	return f.Cleaned
}

func (f *BaseForm) SetInitial(initial map[string]interface{}) {
	f.Initial = initial
}

func (f *BaseForm) SetPrefix(prefix string) {
	f.FormPrefix = prefix
}

func (f *BaseForm) Prefix() string {
	return f.FormPrefix
}

func (f *BaseForm) prefix(name string) string {
	var prefix = f.Prefix()
	if prefix == "" {
		return name
	}
	return fmt.Sprintf("%s-%s", prefix, name)
}

func (f *BaseForm) WithData(data url.Values, files map[string][]io.ReadCloser, r *http.Request) Form {
	f.Reset()
	f.Raw = data
	f.Files = files
	f.setup()
	return f
}

func (f *BaseForm) Reset() {
	f.Raw = nil
	f.Initial = nil
	f.Errors = nil
	f.ErrorList_ = nil
	f.InvalidDefaults = nil
	f.Files = nil
	f.Cleaned = nil
}

func (f *BaseForm) FullClean() {
	f.Errors = orderedmap.NewOrderedMap[string, []error]()

	f.setup()

	if f.Cleaned == nil {
		f.Cleaned = make(map[string]interface{})
	}

	var err error
	for head := f.FormFields.Front(); head != nil; head = head.Next() {
		var (
			k       = head.Key
			v       = head.Value
			errors  []error
			initial interface{}
			data    interface{}
		)

		if v.ReadOnly() {
			continue
		}

		if !v.Widget().ValueOmittedFromData(f.Raw, f.Files, f.prefix(k)) {
			initial, errors = v.Widget().ValueFromDataDict(f.Raw, f.Files, f.prefix(k))
		}

		if len(errors) > 0 {
			f.AddError(k, errors...)
			f.InvalidDefaults[k] = initial
			continue
		}

		if v.Required() && v.IsEmpty(initial) {
			f.AddError(k, errs.NewValidationError(k, errs.ErrFieldRequired))
			f.InvalidDefaults[k] = initial
			continue
		}

		data, err = v.ValueToGo(initial)
		if err != nil {
			f.AddError(k, err)
			f.InvalidDefaults[k] = initial
			continue
		}

		// Set the initial value again in case the value was modified by ValueToGo.
		// This is important so we add the right value to the invalid defaults.
		initial = data

		data, err = v.Clean(initial)
		if err != nil {
			f.AddError(k, err)
			f.InvalidDefaults[k] = initial
			continue
		}

		errors = v.Validate(data)
		if len(errors) > 0 {
			var errList = make([]error, 0, len(errors))
			for _, err := range errors {
				switch e := err.(type) {
				case interface{ Unwrap() []error }:
					errList = append(errList, e.Unwrap()...)
				default:
					errList = append(errList, err)
				}
			}

			f.AddError(k, errList...)
			f.InvalidDefaults[k] = data
			continue
		}

		f.Initial[k] = data
		f.Cleaned[k] = data
	}
}

func (f *BaseForm) Ordering(order []string) {
	f.fieldOrder = order
}

func (f *BaseForm) SetValidators(validators ...func(Form) []error) {
	if f.Validators == nil {
		f.Validators = make([]func(Form) []error, 0)
	}
	f.Validators = append(f.Validators, validators...)
}

func (f *BaseForm) Validate() {
	if f.Validators == nil {
		f.Validators = make([]func(Form) []error, 0)
	}

	for _, validator := range f.Validators {
		var errors = validator(f)
		if len(errors) > 0 {
			for _, err := range errors {
				switch e := err.(type) {
				case interface{ Unwrap() []error }:
					f.AddFormError(e.Unwrap()...)
				default:
					f.AddFormError(e)
				}
			}
		}
	}
}

func (f *BaseForm) IsValid() bool {
	assert.False(f.Raw == nil, "You cannot call IsValid() without setting the data first.")

	if f.Errors == nil {
		f.Errors = orderedmap.NewOrderedMap[string, []error]()
	}

	if f.ErrorList_ == nil {
		f.ErrorList_ = make([]error, 0)
	}

	if f.Cleaned == nil {
		f.FullClean()
	}

	if f.Errors.Len() == 0 {
		f.Validate()
	}

	var valid bool
	if (f.Errors.Len() > 0 || len(f.ErrorList_) > 0) && f.Cleaned != nil {
		f.Cleaned = nil
		valid = false
	} else {
		valid = f.Errors.Len() == 0 && len(f.ErrorList_) == 0
	}

	fmt.Println("IsValid", valid, f.Errors, f.ErrorList_)
	fmt.Println("IsValid", valid, f.Errors, f.ErrorList_)
	fmt.Println("IsValid", valid, f.Errors, f.ErrorList_)
	fmt.Println("IsValid", valid, f.Errors, f.ErrorList_)

	if valid {
		for _, fn := range f.OnValidFuncs {
			fn(f)
		}
	} else {
		for _, fn := range f.OnInvalidFuncs {
			fn(f)
		}
	}

	for _, fn := range f.OnFinalizeFuncs {
		fn(f)
	}

	return valid
}

func (f *BaseForm) AddFormError(errorList ...error) {
	if f.ErrorList_ == nil {
		f.ErrorList_ = make([]error, 0)
	}

	var newErrs = slices.Clone(errorList)
	f.ErrorList_ = append(f.ErrorList_, newErrs...)
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

	f.Errors.Set(name, head)
}

func (f *BaseForm) HasChanged() bool {
	var changed bool = false
	for head := f.FormWidgets.Front(); head != nil; head = head.Next() {
		var (
			k = head.Key
			v = head.Value
		)

		if _, ok := f.Initial[k]; !ok {
			changed = v.ValueOmittedFromData(f.Raw, f.Files, f.prefix(k))
			// If the value is omitted from the data, we don't consider it to have changed.
			return !changed
		}

		if f.Initial[k] != f.Cleaned[k] {
			changed = true
			return false
		}
	}

	return changed
}
