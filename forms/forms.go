package forms

import (
	"html/template"
	"strings"

	"github.com/Nigel2392/router/v3/request"
)

func NewValue(s string) *FormData {
	return &FormData{Val: s}
}

type Form struct {
	Fields      []FormElement
	Errors      FormErrors
	BeforeValid func(*request.Request, *Form) error
	AfterValid  func(*request.Request, *Form) error
}

func (f *Form) Validate() bool {
	var valid = true
	if f.Errors == nil {
		f.Errors = make(FormErrors, 0)
	}
	for _, field := range f.Fields {
		var err = field.Validate()
		if err != nil {
			valid = false
			f.Errors = append(f.Errors, FormError{
				Name:     field.GetName(),
				FieldErr: err,
			})
			field.AddError(err)
		}
	}
	return valid
}

func (f Form) AsP() template.HTML {
	var b strings.Builder
	for _, field := range f.Fields {
		if !field.HasLabel() {
			b.WriteString(`<p>`)
			b.WriteString(field.Label().String())
			b.WriteString("</p>")
		}
		b.WriteString(`<p>`)
		b.WriteString(field.Field().String())
		b.WriteString("</p>")
	}
	return template.HTML(b.String())
}

func (f *Form) Fill(r *request.Request) bool {
	var err error
	r.Request.ParseForm()

	switch r.Method() {
	case "GET", "HEAD", "DELETE":
		f.fillQueries(r)
	case "POST", "PUT", "PATCH":
		f.fillForm(r)
	}

	if f.BeforeValid != nil {
		err = f.BeforeValid(r, f)
		if err != nil {
			f.AddError("BeforeValid", err)
			return false
		}
	}

	valid := f.Validate()

	if f.AfterValid != nil && valid {
		err = f.AfterValid(r, f)
		if err != nil {
			f.AddError("BeforeValid", err)
			return false
		}
	}

	return valid
}

func (f *Form) fillQueries(r *request.Request) {
	for _, field := range f.Fields {
		field.SetValue(r.Request.Form.Get(field.GetName()))
	}
}

func (f *Form) fillForm(r *request.Request) {
	for _, field := range f.Fields {
		if field.IsFile() {

		}
		field.SetValue(r.Request.PostForm.Get(field.GetName()))
	}
}

func (f *Form) Clear() {
	for _, field := range f.Fields {
		field.Clear()
	}
}

func (f *Form) Field(name string) FormElement {
	for _, field := range f.Fields {
		if field.GetName() == name {
			return field
		}
	}
	return nil
}

// AddField adds a field to the form
func (f *Form) AddFields(field ...FormElement) {
	if f.Fields == nil {
		f.Fields = make([]FormElement, 0)
	}
	f.Fields = append(f.Fields, field...)
}

// AddError adds an error to the form
func (f *Form) AddError(name string, err error) {
	if f.Errors == nil {
		f.Errors = make(FormErrors, 0)
	}
	f.Errors = append(f.Errors, FormError{
		Name:     name,
		FieldErr: err,
	})
}

func (f *Form) Without(names ...string) {
	var fields = make([]FormElement, 0)
	for _, field := range f.Fields {
		var found = false
		for _, name := range names {
			if strings.EqualFold(field.GetName(), name) {
				found = true
				break
			}
		}
		if !found {
			fields = append(fields, field)
		}
	}
	f.Fields = fields
}

func (f *Form) Disabled(names ...string) Form {
	if len(names) == 0 {
		for _, field := range f.Fields {
			field.SetDisabled(true)
		}
		return *f
	}
	for _, field := range f.Fields {
		for _, name := range names {
			if strings.EqualFold(field.GetName(), name) {
				field.SetDisabled(true)
				break
			}
		}
	}
	return *f
}

func (f *Form) Get(name string) *FormData {
	for _, field := range f.Fields {
		if field.GetName() == name {
			return field.Value()
		}
	}
	return nil
}
