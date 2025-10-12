package formsets

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"slices"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/Nigel2392/go-django/src/internal/django_reflect"
	"github.com/Nigel2392/go-django/src/utils/mixins"
	"github.com/elliotchance/orderedmap/v2"
)

var (
	_ forms.FullCleanMixin = &ManagementForm{}
)

const (
	TOTAL_FORM_COUNT    = "TOTAL_FORMS"
	INITIAL_FORM_COUNT  = "INITIAL_FORMS"
	MIN_NUM_FORM_COUNT  = "MIN_NUM_FORMS"
	MAX_NUM_FORM_COUNT  = "MAX_NUM_FORMS"
	ORDERING_FIELD_NAME = "__ORDER__"
	DELETION_FIELD_NAME = "__DELETE__"
)

type ManagementForm struct {
	forms.Form

	TotalFormsValue int
	InitialForms    int
	MinNumForms     int
	MaxNumForms     int
}

func NewManagementForm(ctx context.Context, opts ...func(*ManagementForm)) *ManagementForm {
	var m = &ManagementForm{
		Form: forms.NewBaseForm(ctx),
	}
	for _, opt := range opts {
		opt(m)
	}

	m.AddField(TOTAL_FORM_COUNT, fields.NumberField[int](
		fields.Widget(widgets.NewNumberInput[int](nil)),
		fields.Required(true),
		fields.Label("Total Forms"),
		fields.Hide(true),
	))
	m.AddField(INITIAL_FORM_COUNT, fields.CharField(
		fields.Hide(true),
		fields.ReadOnly(true),
	))
	m.AddField(MIN_NUM_FORM_COUNT, fields.CharField(
		fields.Hide(true),
		fields.ReadOnly(true),
	))
	m.AddField(MAX_NUM_FORM_COUNT, fields.CharField(
		fields.Hide(true),
		fields.ReadOnly(true),
	))
	m.SetValidators(func(f forms.Form, data map[string]interface{}) []error {
		var total, ok = data[TOTAL_FORM_COUNT].(int)
		if !ok {
			return []error{errors.TypeMismatch.Wrap(trans.T(
				f.Context(),
				"management form %s must be an integer, got %T", TOTAL_FORM_COUNT, data[TOTAL_FORM_COUNT],
			))}
		}
		if total < m.MinNumForms {
			return []error{errors.ValueError.Wrap(trans.T(
				f.Context(), "Ensure at least %d forms are submitted (you submitted %d).",
				m.MinNumForms, total,
			))}
		}
		if m.MaxNumForms > 0 && total > m.MaxNumForms {
			return []error{errors.ValueError.Wrap(trans.T(
				f.Context(), "Ensure at most %d forms are submitted (you submitted %d).",
				m.MaxNumForms, total,
			))}
		}
		return nil
	})

	return m
}

func (m *ManagementForm) BindCleanedData(invalid, defaults, cleaned map[string]interface{}) {
	m.Form.BindCleanedData(invalid, defaults, cleaned)

	if v, ok := cleaned[TOTAL_FORM_COUNT]; ok {
		if totalForms, ok := v.(int); ok {
			m.TotalFormsValue = totalForms
		}
	}
}

var _ listFormObject[BaseFormSetForm] = (ListFormSet[BaseFormSetForm])(nil)

type ListFormSet[T BaseFormSetForm] interface {
	forms.WithDataDefiner
	Load()
	WithContext(ctx context.Context)
	Context() context.Context
	AddFormError(errorList ...error)
	HasChanged() bool
	Media() media.Media
	Prefix() string
	SetPrefix(prefix string)
	PrefixName(fieldName string) string
	PrefixForm(fld any) string
	DeletedForms() []T
	ManagementForm() *ManagementForm
	Forms() ([]T, error)
	SetForms(forms []T)
	SetDeletedForms(forms []T)
	Field(name string) (fields.Field, bool)
	Widget(name string) (widgets.Widget, bool)
	NewForm(ctx context.Context) T
	Initial(ctx context.Context, totalForms int) (base map[string]interface{}, list []map[string]interface{})
	Form(index int) (form T, ok bool)
	CleanedData() map[string]any // this should probably always return nil
	CleanedDataList() []map[string]any
	ErrorList() []error
	ErrorLists() [][]error
	BoundErrors() *orderedmap.OrderedMap[string, []error]
	BoundErrorsList() []*orderedmap.OrderedMap[string, []error]
	Save() ([]any, error)
}

type BaseFormSetForm interface {
	forms.ErrorDefiner
	forms.WithDataDefiner
	AddFormError(errorList ...error)
	SetPrefix(prefix string)
	Prefix() string
	Field(name string) (fields.Field, bool)
	Widget(name string) (widgets.Widget, bool)
	ErrorList() []error
	BoundErrors() *orderedmap.OrderedMap[string, []error]
	WithContext(ctx context.Context)
	CleanedData() map[string]any
	PrefixName(fieldName string) string
	HasChanged() bool
}

type CleanedDataDefiner interface {
	CleanedData() map[string]interface{}
}

type initialSetter interface {
	SetInitial(initial map[string]interface{})
}

type listFormObject[T any] interface {
	Data() (url.Values, map[string][]filesystem.FileHeader)
	SetForms(forms []T)
	SetDeletedForms(forms []T)
	Forms() ([]T, error)
	PrefixForm(fld any) string
	ManagementForm() *ManagementForm
	Context() context.Context
	WithContext(ctx context.Context)
	ErrorList() []error
	BoundErrors() *orderedmap.OrderedMap[string, []error]
	Initial(ctx context.Context, totalForms int) (base map[string]interface{}, list []map[string]interface{})
	NewForm(ctx context.Context) T
	AddFormError(errors ...error)
}

type formObject[T BaseFormSetForm] struct {
	f T
	i int
	d bool
}

var _ ListFormSet[BaseFormSetForm] = (*BaseFormSet[BaseFormSetForm])(nil)
var _ listFormObject[BaseFormSetForm] = (*BaseFormSet[BaseFormSetForm])(nil)

type FormsetOptions[FORM BaseFormSetForm] struct {
	NewForm          func(c context.Context) FORM
	BeforeCheckValid func(ctx context.Context, formObj any) error
	MinNum           int
	Extra            int
	MaxNum           int
	CanOrder         bool
	CanDelete        bool
	CanAdd           bool
	SkipPrefix       bool
	HideDelete       bool
	DefaultForms     func(ctx context.Context, max, min int) ([]FORM, error)
	DeleteForms      func(ctx context.Context, forms []FORM) error
	GetDefaults      func(ctx context.Context, totalForms int) []map[string]interface{}
	BaseDefaults     func(ctx context.Context) map[string]interface{}
}

type BaseFormSet[FORM BaseFormSetForm] struct {
	opts             FormsetOptions[FORM]
	FormList         []FORM
	DeletedFormsList []FORM
	prefix           string
	ctx              context.Context
	mgmt             *ManagementForm
	validators       []func(FORM, map[string]any) []error
	errors           []error
	req              *http.Request
	formData         url.Values
	formFiles        map[string][]filesystem.FileHeader
}

func NewBaseFormSet[FORM BaseFormSetForm](ctx context.Context, opts FormsetOptions[FORM]) *BaseFormSet[FORM] {
	var mgmt *ManagementForm

	if !((opts.MinNum+opts.Extra) == opts.MaxNum && (opts.MinNum > 0 || !opts.CanAdd)) {
		mgmt = NewManagementForm(ctx, func(m *ManagementForm) {
			m.MinNumForms = opts.MinNum
			m.MaxNumForms = opts.MaxNum
		})
	}

	return &BaseFormSet[FORM]{
		opts: opts,
		mgmt: mgmt,
		ctx:  ctx,
	}
}

func (b *BaseFormSet[FORM]) AddValidator(validator ...func(FORM, map[string]any) []error) {
	b.validators = append(b.validators, validator...)
}

func (b *BaseFormSet[FORM]) AddFormError(errors ...error) {
	b.errors = append(b.errors, errors...)
}

func (b *BaseFormSet[FORM]) Initial(ctx context.Context, totalForms int) (baseDefaults map[string]interface{}, defaultsList []map[string]interface{}) {
	if b.opts.BaseDefaults != nil {
		baseDefaults = b.opts.BaseDefaults(ctx)
	}
	if b.opts.GetDefaults != nil {
		defaultsList = b.opts.GetDefaults(ctx, totalForms)
	}
	return baseDefaults, defaultsList
}

func (b *BaseFormSet[FORM]) Field(name string) (fields.Field, bool) {
	if b.opts.NewForm == nil {
		return nil, false
	}
	var f = b.NewForm(b.ctx)
	return f.Field(name)
}

func (b *BaseFormSet[FORM]) Widget(name string) (widgets.Widget, bool) {
	if b.opts.NewForm == nil {
		return nil, false
	}
	var f = b.NewForm(b.ctx)
	return f.Widget(name)
}

func (b *BaseFormSet[FORM]) ManagementForm() *ManagementForm {
	return b.mgmt
}

func (b *BaseFormSet[FORM]) setupForm(f FORM) {
	var fieldAdder, ok = any(f).(interface {
		AddField(name string, field forms.Field)
	})

	if b.opts.CanOrder {
		if !ok {
			panic("BaseFormSet.setupForm: form does not implement field `AddField(name string, field forms.Field)`")
		}
		fieldAdder.AddField(ORDERING_FIELD_NAME, fields.NumberField[int](
			fields.Required(false),
			fields.Widget(widgets.NewNumberInput[int](nil)),
			fields.Hide(true),
		))
	}

	if b.opts.CanDelete {
		if !ok {
			panic("BaseFormSet.setupForm: form does not implement field `AddField(name string, field forms.Field)`")
		}
		fieldAdder.AddField(DELETION_FIELD_NAME, fields.BooleanField(
			fields.Required(false),
			fields.Label(trans.S("Delete?")),
			fields.Widget(widgets.NewBooleanInput(nil)),
			fields.Hide(b.opts.HideDelete),
		))
	}

	f.WithContext(b.ctx)
}

func (b *BaseFormSet[FORM]) NewForm(ctx context.Context) FORM {
	if b.opts.NewForm == nil {
		panic("FormsetOptions.NewForm is required to create a FormSet")
	}

	var f = b.opts.NewForm(ctx)
	b.setupForm(f)
	return f
}

func (b *BaseFormSet[FORM]) PrefixName(fieldName string) string {
	return b.PrefixForm(fieldName)
}

func (b *BaseFormSet[FORM]) PrefixForm(fieldName any) string {
	if b.prefix != "" {
		return fmt.Sprintf("%s-%v", b.prefix, fieldName)
	}
	return fmt.Sprintf("%v", fieldName)
}

func (b *BaseFormSet[FORM]) Prefix() string {
	return b.prefix
}

func (b *BaseFormSet[FORM]) SetPrefix(prefix string) {
	if b == nil {
		panic("BaseFormSet.SetPrefix: BaseFormSet is nil")
	}

	b.prefix = prefix

	if b.mgmt != nil {
		b.mgmt.SetPrefix(fmt.Sprintf("%s-%s", prefix, "management"))
	}

	if !b.opts.SkipPrefix {
		for i, form := range b.FormList {
			form.SetPrefix(b.PrefixForm(i))
		}
	}
}

func (fs *BaseFormSet[FORM]) Context() context.Context {
	return fs.ctx
}

func (fs *BaseFormSet[FORM]) WithContext(ctx context.Context) {
	fs.ctx = ctx

	for _, form := range fs.FormList {
		form.WithContext(ctx)
	}
}

func (fs *BaseFormSet[FORM]) CheckIsValid(ctx context.Context, formObj any) (isValid bool) {
	form, ok := formObj.(listFormObject[FORM])
	if !ok {
		assert.Fail("BaseFormSet.CheckIsValid only accepts a ListFormSet, got %T", formObj)
	}

	isValid = true
	var data, files = form.Data()
	for mixin := range mixins.Mixins[any](form, true) {
		if prevalidator, ok := mixin.(forms.PrevalidatorMixin); ok {
			var errors = prevalidator.Prevalidate(ctx, formObj, data, files)
			if len(errors) > 0 {
				form.AddFormError(errors...)
			}
		}
	}

	var formList, err = form.Forms() // retrieves the initial forms
	if err != nil {
		assert.Fail("BaseFormSet.CheckIsValid: %v", err)
		return false
	}

	var totalForms int
	var managementForm = form.ManagementForm()
	if managementForm != nil {
		managementForm.WithData(data, files, fs.req)
		managementForm.WithContext(ctx)
		if !forms.IsValid(ctx, managementForm) {
			isValid = false
		}
		totalForms = managementForm.TotalFormsValue
	} else {
		totalForms = len(formList)
	}

	// check if totalforms equals the number of forms
	if !fs.opts.CanAdd && totalForms > len(formList) {
		form.AddFormError(errors.ValueError.Wrap(
			trans.T(ctx, "You cannot add more forms than the maximum allowed."),
		))
		isValid = false
		totalForms = len(formList)
	}

	if len(formList) > totalForms {
		formList = formList[:totalForms]
	}

	var base, defaults = form.Initial(ctx, totalForms)
	var loopIters = max(totalForms, fs.opts.MinNum)
	if fs.opts.CanAdd {
		loopIters = max(loopIters, len(formList))
	}

	var totalDeleted int
	var totalAdded int
	var formObjs = make([]formObject[FORM], 0, loopIters)
	for i := 0; i < loopIters; i++ {
		if totalAdded >= totalForms {
			break
		}

		var subForm FORM
		if fs.opts.CanAdd && totalAdded >= len(formList) {
			subForm = form.NewForm(ctx)
		} else {
			subForm = formList[totalAdded]
		}

		if !fs.opts.SkipPrefix {
			subForm.SetPrefix(form.PrefixForm(i))
		}

		subForm.WithContext(form.Context())
		subForm.WithData(data, files, fs.req)

		var formObj = formObject[FORM]{
			f: subForm,
			i: totalAdded,
			d: false,
		}

		if fs.opts.CanOrder {
			ordering, _, errs := forms.ValueFromDataDict[int](ctx, subForm, ORDERING_FIELD_NAME, data, files)
			if len(errs) > 0 {
				subForm.AddFormError(errs...)
				isValid = false
			}
			formObj.i = ordering
		}

		if fs.opts.CanDelete {
			deletion, _, errs := forms.ValueFromDataDict[bool](ctx, subForm, DELETION_FIELD_NAME, data, files)
			if len(errs) > 0 {
				subForm.AddFormError(errs...)
				isValid = false
			}
			formObj.d = deletion
		}

		formObjs = append(formObjs, formObj)

		if formObj.d {
			totalDeleted++
			continue
		}

		if s, ok := any(subForm).(initialSetter); ok {
			if totalAdded < len(defaults) && defaults[totalAdded] != nil {
				s.SetInitial(defaults[totalAdded])
			} else if base != nil {
				s.SetInitial(base)
			}
		}

		// Increase the totalAdded count only for non-deleted forms
		totalAdded++

		isValid = forms.IsValid(ctx, formObj.f) && isValid

		if def, ok := any(formObj.f).(CleanedDataDefiner); ok && isValid {
			var cleaned = def.CleanedData()
			for _, validator := range fs.validators {
				// then run any additional validators
				// that were added to the formset
				var errors = validator(formObj.f, cleaned)
				if len(errors) > 0 {
					formObj.f.AddFormError(errors...)
				}

				isValid = isValid && len(errors) == 0 && !forms.HasErrors(formObj.f)
			}
		}
	}

	slices.SortStableFunc(formObjs, func(a, b formObject[FORM]) int {
		return a.i - b.i
	})

	var finalForms = make([]FORM, 0, len(formObjs))
	var deletedForms = make([]FORM, 0, totalDeleted)
	for _, formObj := range formObjs {
		if formObj.d {
			deletedForms = append(deletedForms, formObj.f)
			continue
		}
		finalForms = append(finalForms, formObj.f)
	}

	// set the final forms to the formset
	// after ordering and deletion
	form.SetForms(finalForms)

	// set the deleted forms to the formset
	form.SetDeletedForms(deletedForms)

	return isValid && !forms.HasErrors(form)
}

func (b *BaseFormSet[FORM]) Data() (url.Values, map[string][]filesystem.FileHeader) {
	return b.formData, b.formFiles
}

func (b *BaseFormSet[FORM]) WithData(data url.Values, files map[string][]filesystem.FileHeader, r *http.Request) {
	b.req = r
	b.formData = data
	b.formFiles = files
}

func (b *BaseFormSet[FORM]) Load() {
	if (b.formData != nil && b.req != nil || b.formFiles != nil && b.req != nil) || forms.HasErrors(b) {
		logger.Warnf("Formset: already loaded, skipping Load()")
		return
	}

	_, err := b.Forms()
	if err != nil {
		return
	}

	var mgmt = b.ManagementForm()
	if mgmt != nil {
		mgmt.InitialForms = len(b.FormList)
		mgmt.TotalFormsValue = len(b.FormList)
		mgmt.SetInitial(map[string]interface{}{
			TOTAL_FORM_COUNT:   mgmt.TotalFormsValue,
			INITIAL_FORM_COUNT: mgmt.InitialForms,
			MIN_NUM_FORM_COUNT: mgmt.MinNumForms,
			MAX_NUM_FORM_COUNT: mgmt.MaxNumForms,
		})
	}

	for _, form := range b.FormList {
		if loader, ok := any(form).(interface{ Load() }); ok {
			loader.Load()
		}
	}
}

func (b *BaseFormSet[FORM]) HasChanged() bool {
	for _, form := range b.FormList {
		if form.HasChanged() {
			return true
		}
	}

	return false
}

func (b *BaseFormSet[FORM]) Media() media.Media {
	var m media.Media = media.NewMedia()
	if b.opts.NewForm == nil {
		for _, f := range b.FormList {
			if mf, ok := any(f).(media.MediaDefiner); ok {
				m = m.Merge(mf.Media())
			}
		}
		return m
	}
	var f = b.NewForm(b.ctx)
	if m, ok := any(f).(media.MediaDefiner); ok {
		return m.Media()
	}
	return m
}

func (b *BaseFormSet[FORM]) DeletedForms() []FORM {
	return b.DeletedFormsList
}

func (b *BaseFormSet[FORM]) Forms() ([]FORM, error) {
	if b.FormList != nil {
		return b.FormList, nil
	}

	var maxNum = max(b.opts.MaxNum, b.opts.MinNum+b.opts.Extra)
	var minNum = min(b.opts.MinNum+b.opts.Extra, maxNum)
	var forms []FORM
	var err error
	if b.opts.DefaultForms != nil {
		forms, err = b.opts.DefaultForms(b.ctx, maxNum, minNum)
	} else {
		forms = make([]FORM, 0, minNum)
	}
	if err != nil {
		return nil, err
	}

	var base, defaults = b.Initial(b.ctx, maxNum)
	var allForms = make([]FORM, max(minNum, len(forms)))
	for i := 0; i < max(minNum, len(forms)); i++ {
		var form FORM
		if i < len(forms) {
			form = forms[i]
			b.setupForm(form)
		} else {
			form = b.NewForm(b.ctx)
		}

		if s, ok := any(form).(initialSetter); ok {
			if i < len(defaults) && defaults[i] != nil {
				s.SetInitial(defaults[i])
			} else if base != nil {
				s.SetInitial(base)
			}
		}

		allForms[i] = form
	}

	b.SetForms(allForms)

	return b.FormList, nil
}

func (b *BaseFormSet[FORM]) ForEach(fn func(form FORM, index int) error) error {
	var forms, err = b.Forms()
	if err != nil {
		return err
	}
	for i, form := range forms {
		if err := fn(form, i); err != nil {
			return err
		}
	}
	return nil
}

func (b *BaseFormSet[FORM]) SetDeletedForms(forms []FORM) {
	b.DeletedFormsList = forms
}

func (b *BaseFormSet[FORM]) SetForms(forms []FORM) {
	b.FormList = forms

	if b.mgmt != nil {
		b.mgmt.InitialForms = len(forms)
		b.mgmt.TotalFormsValue = len(forms)
		b.mgmt.SetInitial(map[string]interface{}{
			TOTAL_FORM_COUNT:   b.mgmt.TotalFormsValue,
			INITIAL_FORM_COUNT: b.mgmt.InitialForms,
			MIN_NUM_FORM_COUNT: b.mgmt.MinNumForms,
			MAX_NUM_FORM_COUNT: b.mgmt.MaxNumForms,
		})
	}

	for i, form := range b.FormList {
		if !b.opts.SkipPrefix {
			form.SetPrefix(b.PrefixForm(i))
		}
		form.WithContext(b.ctx)
	}
}

func (b *BaseFormSet[FORM]) Form(i int) (form FORM, ok bool) {
	var forms, err = b.Forms()
	if err != nil {
		assert.Fail("BaseFormSet.Form: %v", err)
		return *new(FORM), false
	}

	if i < 0 || i >= len(forms) {
		return *new(FORM), false
	}
	return forms[i], true
}

func (b *BaseFormSet[FORM]) CleanedData() map[string]any { // []map[string]any {
	//	var cleaned = make([]map[string]any, len(b.FormList))
	//	for i, form := range b.FormList {
	//		cleaned[i] = form.CleanedData()
	//	}
	//	return cleaned
	return nil
}

func (b *BaseFormSet[FORM]) CleanedDataList() []map[string]any {
	var cleaned = make([]map[string]any, len(b.FormList))
	for i, form := range b.FormList {
		cleaned[i] = form.CleanedData()
	}
	return cleaned
}

func (b *BaseFormSet[FORM]) ErrorList() []error {
	var errs = make([]error, 0, len(b.errors)+len(b.FormList))
	errs = append(errs, b.errors...)
	for _, form := range b.FormList {
		errs = append(errs, form.ErrorList()...)
	}
	return errs
}

func (b *BaseFormSet[FORM]) BoundErrors() *orderedmap.OrderedMap[string, []error] {
	var baseErrs = orderedmap.NewOrderedMap[string, []error]()
	for _, form := range b.FormList {
		var errs = form.BoundErrors()
		if errs == nil || errs.Len() == 0 {
			continue
		}

		for head := errs.Front(); head != nil; head = head.Next() {
			var errsList, _ = baseErrs.Get(head.Key)
			errsList = append(errsList, head.Value...)
			baseErrs.Set(head.Key, errsList)
		}
	}
	return baseErrs
}

func (b *BaseFormSet[FORM]) ErrorLists() [][]error {
	var errs = make([][]error, 0, len(b.FormList))
	for _, form := range b.FormList {
		errs = append(errs, form.ErrorList())
	}
	return errs
}

func (b *BaseFormSet[FORM]) BoundErrorsList() []*orderedmap.OrderedMap[string, []error] {
	var errs = make([]*orderedmap.OrderedMap[string, []error], 0, len(b.FormList))
	for _, form := range b.FormList {
		errs = append(errs, form.BoundErrors())
	}
	return errs
}

func (b *BaseFormSet[FORM]) Save() ([]any, error) {
	var results = make([]any, 0, len(b.FormList))
	var errs = make([]error, 0)
	for _, form := range b.FormList {
		var cleaned = form.CleanedData()
		var isDeleted = false
		if del, ok := cleaned[DELETION_FIELD_NAME]; ok {
			isDeleted, _ = del.(bool)
		}
		if isDeleted {
			continue
		}

		var wrapFn = django_reflect.WrapWithContext(b.ctx)
		var rv = reflect.ValueOf(form)
		var saveMethod = rv.MethodByName("Save")
		switch saveMethod.Type().NumOut() {
		case 1:
			var saveMethod, err = django_reflect.Method[func() error](form, "Save", wrapFn)
			if err != nil {
				return nil, errors.Wrapf(err, "form %T does not have a Save method", form)
			}

			err = saveMethod()
			if err != nil {
				errs = append(errs, err)
				continue
			}
		case 2:
			var saveMethod, err = django_reflect.Method[func() (any, error)](form, "Save", wrapFn)
			if err != nil {
				return nil, errors.Wrapf(err, "form %T does not have a Save method", form)
			}
			i, err := saveMethod()
			if err != nil {
				errs = append(errs, err)
				continue
			}
			results = append(results, i)
		default:
			assert.Fail("form %T Save method must return one or two values, got %d", form, saveMethod.Type().NumOut())
		}
	}

	if b.opts.DeleteForms != nil && len(b.DeletedFormsList) > 0 {
		if err := b.opts.DeleteForms(b.ctx, b.DeletedFormsList); err != nil {
			return nil, err
		}
	}

	return results, nil
}
