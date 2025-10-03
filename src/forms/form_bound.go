package forms

import (
	"bytes"
	"context"
	"html/template"

	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/elliotchance/orderedmap/v2"
)

type ErrorUnpacker interface {
	UnpackErrors(boundForm BoundForm, boundErrors *orderedmap.OrderedMap[string, []error]) []FieldError
}

type fieldError struct {
	name   string
	field  string
	errors []error
}

func (f *fieldError) Field() string {
	return f.field
}

func (f *fieldError) Name() string {
	return f.name
}

func (f *fieldError) Errors() []error {
	return f.errors
}

type BaseBoundForm struct {
	Form          FormBinder
	BoundFields   []BoundField
	BoundFieldMap map[string]BoundField
	ErrorMap      *orderedmap.OrderedMap[string, []error]
	ListErrors    []error
	media         media.Media
	renderer      FormRenderer
	context       context.Context
}

type FormBinder interface {
	FieldOrder() []string
	Context() context.Context
	BoundFields() *orderedmap.OrderedMap[string, BoundField]
	ErrorDefiner
}

func NewBoundForm(ctx context.Context, f FormBinder, renderer FormRenderer) BoundForm {
	var (
		fields      = f.BoundFields()
		errors      = f.BoundErrors()
		boundFields = make([]BoundField, 0, fields.Len())
		fieldMap    = make(map[string]BoundField, fields.Len())
	)

	if renderer == nil {
		renderer = &defaultRenderer{}
	}

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
		renderer:      renderer,
		context:       ctx,
	}
}

func (f *BaseBoundForm) AsP() template.HTML {
	if f, ok := f.Form.(interface{ AsP() template.HTML }); ok {
		return f.AsP()
	}
	var b = new(bytes.Buffer)
	err := f.renderer.RenderAsP(b, f.context, f)
	if err != nil {
		return template.HTML("")
	}
	return template.HTML(b.String())
}

func (f *BaseBoundForm) AsUL() template.HTML {
	if f, ok := f.Form.(interface{ AsUL() template.HTML }); ok {
		return f.AsUL()
	}
	var b = new(bytes.Buffer)
	err := f.renderer.RenderAsUL(b, f.context, f)
	if err != nil {
		return template.HTML("")
	}
	return template.HTML(b.String())
}

func (f *BaseBoundForm) AsTable() template.HTML {
	if f, ok := f.Form.(interface{ AsTable() template.HTML }); ok {
		return f.AsTable()
	}
	var b = new(bytes.Buffer)
	err := f.renderer.RenderAsTable(b, f.context, f)
	if err != nil {
		return template.HTML("")
	}
	return template.HTML(b.String())
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

func UnpackErrors[T interface{ Label(context.Context) string }](bf BoundForm, f FormBinder, errorMap *orderedmap.OrderedMap[string, []error], getLabel func(string) (T, bool)) []FieldError {
	if unpacker, ok := f.(ErrorUnpacker); ok {
		return unpacker.UnpackErrors(bf, errorMap)
	}

	if errorMap == nil {
		return nil
	}

	var ret = make([]FieldError, 0, errorMap.Len())
	for head := errorMap.Front(); head != nil; head = head.Next() {
		var err = fieldError{
			name:   head.Key,
			field:  head.Key,
			errors: head.Value,
		}

		var field, ok = getLabel(head.Key)
		if ok {
			err.field = field.Label(f.Context())
		}

		ret = append(ret, &err)
	}
	return ret

}

func (f *BaseBoundForm) UnpackErrors() []FieldError {
	return UnpackErrors(f, f.Form, f.ErrorMap, func(s string) (Field, bool) {
		fld, ok := f.BoundFieldMap[s]
		if !ok {
			return nil, false
		}
		return fld.Input(), true
	})
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
