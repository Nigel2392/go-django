package filters

import (
	"errors"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/elliotchance/orderedmap/v2"
)

type FilterSpec[T attrs.Definer] interface {
	// Name returns the name of the filter.
	Name() string

	// Label returns an optional label for the filter.
	// If not provided, the name will be used.
	// This is commented out due to it not being required.
	// Label() string

	// Field returns the field of the filter.
	Field() fields.Field

	// Filter filters the object list.
	Filter(value interface{}, object *queries.QuerySet[T]) (*queries.QuerySet[T], error)
}

type Filters[T attrs.Definer] struct {
	form  *forms.BaseForm
	specs *orderedmap.OrderedMap[string, FilterSpec[T]]
}

func NewFilters[T attrs.Definer](formPrefix string, specs ...FilterSpec[T]) *Filters[T] {
	var s = orderedmap.NewOrderedMap[string, FilterSpec[T]]()
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

	return &Filters[T]{
		specs: s,
		form:  form,
	}
}

func (f *Filters[T]) Add(spec FilterSpec[T]) {
	f.specs.Set(spec.Name(), spec)
}

func (f *Filters[T]) Specs() *orderedmap.OrderedMap[string, FilterSpec[T]] {
	return f.specs
}

func (f *Filters[T]) Form() *forms.BaseForm {
	return f.form
}

var FormError = errors.New("form is not valid")

func (f *Filters[T]) Filter(data map[string][]string, object *queries.QuerySet[T]) (*queries.QuerySet[T], error) {
	f.form.Reset()

	f.form.FormFields = orderedmap.NewOrderedMap[string, fields.Field]()
	f.form.Errors = orderedmap.NewOrderedMap[string, []error]()

	for front := f.specs.Front(); front != nil; front = front.Next() {
		var spec = front.Value
		var field = spec.Field()
		f.form.AddField(spec.Name(), field)
	}

	f.form.WithData(
		data, nil, nil,
	)

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

		errList = append(
			errList,
			FormError,
		)

		return object, errors.Join(errList...)
	}

	var cleanedData = f.form.CleanedData()
	for front := f.specs.Front(); front != nil; front = front.Next() {
		var spec = front.Value
		var value, ok = cleanedData[spec.Name()]
		if !ok {
			continue
		}

		var err error
		object, err = spec.Filter(value, object)
		if err != nil {
			return nil, errors.Join(
				err,
				errors.New("filtering failed for: "+spec.Name()),
			)
		}
	}

	return object, nil
}
