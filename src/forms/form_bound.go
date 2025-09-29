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

type BaseBoundForm struct {
	Form          any
	BoundFields   []BoundField
	BoundFieldMap map[string]BoundField
	ErrorMap      *orderedmap.OrderedMap[string, []error]
	ListErrors    []error
	media         media.Media
}

type FormBinder interface {
	FieldOrder() []string
	BoundFields() *orderedmap.OrderedMap[string, BoundField]
	ErrorDefiner
}

func NewBoundForm(f FormBinder) BoundForm {
	var (
		fields      = f.BoundFields()
		errors      = f.BoundErrors()
		boundFields = make([]BoundField, 0, fields.Len())
		fieldMap    = make(map[string]BoundField, fields.Len())
	)
	//for head := fields.Front(); head != nil; head = head.Next() {
	//	boundFields = append(boundFields, head.Value)
	//}
	var fieldOrder = f.FieldOrder()
	if fieldOrder != nil {
		var had = make(map[string]struct{})
		for _, k := range fieldOrder {
			if v, ok := fields.Get(k); ok {
				boundFields = append(boundFields, v)
				fieldMap[k] = v
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
					fieldMap[k] = v
				}
			}
		}
	} else {
		for head := fields.Front(); head != nil; head = head.Next() {
			boundFields = append(boundFields, head.Value)
			fieldMap[head.Key] = head.Value
		}
	}

	return &BaseBoundForm{
		Form:          f,
		BoundFields:   boundFields,
		BoundFieldMap: fieldMap,
		ErrorMap:      errors,
		ListErrors:    f.ErrorList(),
	}
}

func (f *BaseBoundForm) AsP() template.HTML {
	if f, ok := f.Form.(interface{ AsP() template.HTML }); ok {
		return f.AsP()
	}
	return template.HTML("")
}

func (f *BaseBoundForm) AsUL() template.HTML {
	if f, ok := f.Form.(interface{ AsUL() template.HTML }); ok {
		return f.AsUL()
	}
	return template.HTML("")
}

func (f *BaseBoundForm) Fields() []BoundField {
	return f.BoundFields
}

func (f *BaseBoundForm) FieldMap() map[string]BoundField {
	return f.BoundFieldMap
}

func (f *BaseBoundForm) Errors() *orderedmap.OrderedMap[string, []error] {
	return f.ErrorMap
}

func (f *BaseBoundForm) UnpackErrors() []FieldError {
	if f.ErrorMap == nil {
		return nil
	}
	var ret = make([]FieldError, 0, f.ErrorMap.Len())
	for head := f.ErrorMap.Front(); head != nil; head = head.Next() {
		ret = append(ret, &fieldError{
			field:  head.Key,
			errors: head.Value,
		})
	}
	return ret
}

func (f *BaseBoundForm) ErrorList() []error {
	return f.ListErrors
}

func (f *BaseBoundForm) Media() media.Media {
	if f.media == nil {
		if fm, ok := f.Form.(media.MediaDefiner); ok {
			f.media = fm.Media()
		}
	}
	return f.media
}
