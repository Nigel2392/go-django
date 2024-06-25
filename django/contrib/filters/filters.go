package filters

import (
	"errors"

	"github.com/Nigel2392/django/forms"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/elliotchance/orderedmap/v2"
)

type FilterSpec[ListT any] interface {
	// Name returns the name of the filter.
	Name() string

	// Label returns an optional label for the filter.
	// If not provided, the name will be used.
	// This is commented out due to it not being required.
	// Label() string

	// Field returns the field of the filter.
	Field() fields.Field

	// Filter filters the object list.
	Filter(value interface{}, objectList []ListT)
}

type Filters[ListT any] struct {
	form  *forms.BaseForm
	specs *orderedmap.OrderedMap[string, FilterSpec[ListT]]
}

func NewFilters[FormValueT, ListT any](formPrefix string, specs ...FilterSpec[ListT]) *Filters[ListT] {
	var s = orderedmap.NewOrderedMap[string, FilterSpec[ListT]]()
	for _, spec := range specs {
		s.Set(spec.Name(), spec)
	}

	var opts = make([]func(forms.Form), 0)
	if formPrefix != "" {
		opts = append(opts, forms.WithPrefix(formPrefix))
	}

	var form = forms.NewBaseForm(
		opts...,
	)

	return &Filters[ListT]{
		specs: s,
		form:  form,
	}
}

func (f *Filters[ListT]) Add(spec FilterSpec[ListT]) {
	f.specs.Set(spec.Name(), spec)
}

func (f *Filters[ListT]) Specs() *orderedmap.OrderedMap[string, FilterSpec[ListT]] {
	return f.specs
}

func (f *Filters[ListT]) Form() *forms.BaseForm {
	return f.form
}

func (f *Filters[ListT]) Filter(data map[string][]string, objects []ListT) error {
	f.form.Reset()

	f.form.FormFields = orderedmap.NewOrderedMap[string, fields.Field]()
	f.form.Errors = orderedmap.NewOrderedMap[string, []error]()

	f.form.WithData(
		data, nil, nil,
	)

	for front := f.specs.Front(); front != nil; front = front.Next() {
		var spec = front.Value
		var field = spec.Field()
		f.form.AddField(spec.Name(), field)
	}

	if !f.form.IsValid() {
		var errs = f.form.Errors
		var errList = make([]error, 0, errs.Len())

		for front := errs.Front(); front != nil; front = front.Next() {
			errList = append(
				errList,
				errors.Join(front.Value...),
			)
		}

		errList = append(
			errList,
			errors.Join(f.form.ErrorList_...),
		)

		return errors.Join(errList...)
	}

	var cleanedData = f.form.CleanedData()
	for front := f.specs.Front(); front != nil; front = front.Next() {
		var spec = front.Value
		var value, ok = cleanedData[spec.Name()]
		if !ok {
			continue
		}

		spec.Filter(value, objects)
	}

	return nil
}
