package formsets

import (
	"context"
	"net/http"
	"strconv"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/elliotchance/orderedmap/v2"
)

var (
	_ forms.PrevalidatorMixin = &ManagementForm{}
	_ forms.ValidatorMixin    = &ManagementForm{}
	_ forms.FullCleanMixin    = &ManagementForm{}
)

const (
	TOTAL_FORM_COUNT    = "TOTAL_FORMS"
	INITIAL_FORM_COUNT  = "INITIAL_FORMS"
	MIN_NUM_FORM_COUNT  = "MIN_NUM_FORMS"
	MAX_NUM_FORM_COUNT  = "MAX_NUM_FORMS"
	ORDERING_FIELD_NAME = "ORDER"
	DELETION_FIELD_NAME = "DELETE"
)

type ManagementForm struct {
	Prefix       string
	TotalForms   int
	InitialForms int
	MinNumForms  int
	MaxNumForms  int

	fieldMap *orderedmap.OrderedMap[string, forms.Field]
}

func (m *ManagementForm) PrefixName(fieldName string) string {
	if m.Prefix != "" {
		var b = make([]byte, 0, len(m.Prefix)+len(fieldName)+1)
		b = append(b, m.Prefix...)
		b = append(b, '-')
		b = append(b, fieldName...)
		return string(b)
	}
	return fieldName
}

func (m *ManagementForm) Prevalidate(ctx context.Context, root forms.Form) []error {
	var data, _ = root.Data()
	if data == nil {
		return nil
	}

	totalFormsStr, ok := data[m.PrefixName(TOTAL_FORM_COUNT)]
	if !ok || len(totalFormsStr) == 0 {
		return []error{errors.ValueError.Wrap(trans.T(
			ctx, "Management form is missing %s field.",
			TOTAL_FORM_COUNT,
		))}
	}

	initialFormsStr, ok := data[m.PrefixName(INITIAL_FORM_COUNT)]
	if !ok || len(initialFormsStr) == 0 {
		return []error{errors.ValueError.Wrap(trans.T(
			ctx, "Management form is missing %s field.",
			INITIAL_FORM_COUNT,
		))}
	}

	totalForms, err := strconv.Atoi(totalFormsStr[0])
	if err != nil || totalForms < 0 {
		return []error{errors.ValueError.Wrap(trans.T(
			ctx, "Invalid value for %s field.",
			TOTAL_FORM_COUNT,
		))}
	}

	initialForms, err := strconv.Atoi(initialFormsStr[0])
	if err != nil || initialForms < 0 {
		return []error{errors.ValueError.Wrap(trans.T(
			ctx, "Invalid value for %s field.",
			INITIAL_FORM_COUNT,
		))}
	}

	m.TotalForms = totalForms
	m.InitialForms = initialForms
	return nil
}

func (m *ManagementForm) BindCleanedData(invalid, defaults, cleaned map[string]interface{}) {
	// safe to be a no-op, as [ManagementForm] should always be valid after Prevalidate
}

func (m *ManagementForm) FieldMap() *orderedmap.OrderedMap[string, forms.Field] {
	if m.fieldMap != nil {
		return m.fieldMap
	}

	m.fieldMap = orderedmap.NewOrderedMap[string, forms.Field]()
	m.fieldMap.Set(TOTAL_FORM_COUNT, fields.CharField(
		fields.Hide(true),
		fields.Required(true),
	))
	m.fieldMap.Set(INITIAL_FORM_COUNT, fields.CharField(
		fields.Hide(true),
		fields.Required(true),
	))
	m.fieldMap.Set(MIN_NUM_FORM_COUNT, fields.CharField(
		fields.Hide(true),
		fields.ReadOnly(true),
	))
	m.fieldMap.Set(MAX_NUM_FORM_COUNT, fields.CharField(
		fields.Hide(true),
		fields.ReadOnly(true),
	))
	return m.fieldMap
}

func (m *ManagementForm) Widget(name string) (forms.Widget, bool) {
	f, ok := m.FieldMap().Get(name)
	if !ok {
		return nil, false
	}
	return f.Widget(), true
}

func (m *ManagementForm) Validators() []func(ctx context.Context, root forms.Form) []error {
	return []func(ctx context.Context, root forms.Form) []error{
		func(ctx context.Context, root forms.Form) []error {
			if m.TotalForms < m.MinNumForms {
				return []error{errors.ValueError.Wrap(trans.T(
					ctx, "Ensure at least %d forms are submitted (you submitted %d).",
					m.MinNumForms, m.TotalForms,
				))}
			}
			if m.MaxNumForms > 0 && m.TotalForms > m.MaxNumForms {
				return []error{errors.ValueError.Wrap(trans.T(
					ctx, "Ensure at most %d forms are submitted (you submitted %d).",
					m.MaxNumForms, m.TotalForms,
				))}
			}
			return nil
		},
	}
}

type ListFormSet interface {
	forms.FormWrapper
	Context() context.Context
	HasChanged() bool
	Media() media.Media
	SetPrefix(prefix string)
	Form(index int) (form forms.Form, ok bool)
	CleanedData() []map[string]any
	ErrorList() [][]error
	BoundErrors() []*orderedmap.OrderedMap[string, []error]
}

var _ ListFormSet = (*BaseFormSet[forms.Form])(nil)

type BaseFormSet[FORM forms.Form] struct {
	ManagementForm *ManagementForm
	Prefix         string
	ctx            context.Context
	forms          []FORM
	validators     []func([]FORM, []map[string]any) []error
}

func (b *BaseFormSet[FORM]) SetPrefix(prefix string) {
	b.Prefix = prefix
	b.ManagementForm.Prefix = prefix

	for i, form := range b.forms {
		var prefix = b.Prefix
		if prefix == "" {
			prefix = strconv.Itoa(i)
		}

		form.SetPrefix(prefix)
		b.forms[i] = form
	}
}

func (b *BaseFormSet[FORM]) WithData(data map[string][]string, files map[string][]filesystem.FileHeader, r *http.Request) ListFormSet {
	for i, form := range b.forms {
		b.forms[i] = form.WithData(data, files, r).(FORM)
	}
	return b
}

func (b *BaseFormSet[FORM]) Context() context.Context {
	return b.ctx
}

func (b *BaseFormSet[FORM]) WithContext(ctx context.Context) {
	b.ctx = ctx
	for _, form := range b.forms {
		form.WithContext(ctx)
	}
}

func (b *BaseFormSet[FORM]) HasChanged() bool {
	for _, form := range b.forms {
		if form.HasChanged() {
			return true
		}
	}

	return false
}

func (b *BaseFormSet[FORM]) Media() media.Media {
	var m media.Media = media.NewMedia()
	for _, form := range b.forms {
		m = m.Merge(form.Media())
	}
	return m
}

func (b *BaseFormSet[FORM]) Unwrap() []forms.Form {
	var fs = make([]forms.Form, len(b.forms))
	for i, f := range b.forms {
		fs[i] = f
	}
	return fs
}

func (b *BaseFormSet[FORM]) Forms() []FORM {
	return b.forms
}

func (b *BaseFormSet[FORM]) Form(i int) (form forms.Form, ok bool) {
	if i < 0 || i >= len(b.forms) {
		return nil, false
	}
	return b.forms[i], true
}

func (b *BaseFormSet[FORM]) CleanedData() []map[string]any {
	var cleaned = make([]map[string]any, len(b.forms))
	for i, form := range b.forms {
		cleaned[i] = form.CleanedData()
	}
	return cleaned
}

func (b *BaseFormSet[FORM]) ErrorList() [][]error {
	var errs = make([][]error, len(b.forms))
	for i, form := range b.forms {
		errs[i] = form.ErrorList()
	}
	return errs
}

func (b *BaseFormSet[FORM]) BoundErrors() []*orderedmap.OrderedMap[string, []error] {
	var errs = make([]*orderedmap.OrderedMap[string, []error], len(b.forms))
	for i, form := range b.forms {
		errs[i] = form.BoundErrors()
	}
	return errs
}
