package formsets

import (
	"context"

	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/elliotchance/orderedmap/v2"
)

type ListFormSet interface {
	Context() context.Context
	HasChanged() bool
	IsValid() bool
	Media() media.Media
	Forms() []forms.Form
	Form(index int) (form forms.Form, ok bool)
	CleanedData() []map[string]any
	ErrorList() [][]error
	BoundErrors() []*orderedmap.OrderedMap[string, []error]
}

var _ ListFormSet = (*BaseFormSet)(nil)

type BaseFormSet struct {
	formContext context.Context
	forms       []forms.Form
	validators  []func([]forms.Form, []map[string]any) []error
}

func (b *BaseFormSet) Context() context.Context {
	return b.formContext
}

func (b *BaseFormSet) WithContext(ctx context.Context) {
	b.formContext = ctx
	for _, form := range b.forms {
		form.WithContext(ctx)
	}
}

func (b *BaseFormSet) HasChanged() bool {
	for _, form := range b.forms {
		if form.HasChanged() {
			return true
		}
	}

	return false
}

func (b *BaseFormSet) IsValid() bool {
	for _, form := range b.forms {
		if !forms.IsValid(form.Context(), form) {
			return false
		}
	}

	return true
}

func (b *BaseFormSet) Media() media.Media {
	var m media.Media = media.NewMedia()
	for _, form := range b.forms {
		m = m.Merge(form.Media())
	}
	return m
}

func (b *BaseFormSet) Forms() []forms.Form {
	return b.Forms()
}

func (b *BaseFormSet) Form(i int) (form forms.Form, ok bool) {
	if i < 0 || i >= len(b.forms) {
		return nil, false
	}
	return b.forms[i], true
}

func (b *BaseFormSet) CleanedData() []map[string]any {
	var cleaned = make([]map[string]any, len(b.forms))
	for i, form := range b.forms {
		cleaned[i] = form.CleanedData()
	}
	return cleaned
}

func (b *BaseFormSet) ErrorList() [][]error {
	var errs = make([][]error, len(b.forms))
	for i, form := range b.forms {
		errs[i] = form.ErrorList()
	}
	return errs
}

func (b *BaseFormSet) BoundErrors() []*orderedmap.OrderedMap[string, []error] {
	var errs = make([]*orderedmap.OrderedMap[string, []error], len(b.forms))
	for i, form := range b.forms {
		errs[i] = form.BoundErrors()
	}
	return errs
}
