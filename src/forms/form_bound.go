package forms

import (
	"html/template"

	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/elliotchance/orderedmap/v2"
)

type fieldError struct {
	field  string
	errors []error
}

func (f *fieldError) Field() string {
	return f.field
}

func (f *fieldError) Errors() []error {
	return f.errors
}

type BoundForm interface {
	AsP() template.HTML
	AsUL() template.HTML
	Media() media.Media
	Fields() []BoundField
	ErrorList() []error
	UnpackErrors() []FieldError
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

func (f *_BoundForm) UnpackErrors() []FieldError {
	if f.Errors_ == nil {
		return nil
	}
	var ret = make([]FieldError, 0, f.Errors_.Len())
	for head := f.Errors_.Front(); head != nil; head = head.Next() {
		ret = append(ret, &fieldError{
			field:  head.Key,
			errors: head.Value,
		})
	}
	return ret
}

func (f *_BoundForm) ErrorList() []error {
	return f.ErrorList_
}

func (f *_BoundForm) Media() media.Media {
	return f.Form.Media()
}
