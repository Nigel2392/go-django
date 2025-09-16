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

type _BoundForm struct {
	Form       Form
	Fields_    []BoundField
	FieldMap_  map[string]BoundField
	Errors_    *orderedmap.OrderedMap[string, []error]
	ErrorList_ []error
	media      media.Media
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

func (f *_BoundForm) FieldMap() map[string]BoundField {
	return f.FieldMap_
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
	if f.media == nil {
		f.media = f.Form.Media()
	}
	return f.media
}
