package forms

import (
	"html/template"
	"strings"

	"github.com/Nigel2392/router/v3/request"
)

type Form struct {
	Fields      []*Field
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
				Name:     field.Name,
				FieldErr: err,
			})
			field.Errors.Add(field.Name, err)
		}
	}
	return valid
}

func (f Form) AsP() template.HTML {
	var b strings.Builder
	for _, field := range f.Fields {
		if field.LabelText != "" {
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
		field.Value = r.Request.Form.Get(field.Name)
	}
}

func (f *Form) fillForm(r *request.Request) {
	for _, field := range f.Fields {
		field.Value = r.Request.PostForm.Get(field.Name)
	}
}

func (f *Form) Clear() {
	for _, field := range f.Fields {
		field.Value = ""
	}
}

func (f *Form) Field(name string) *Field {
	for _, field := range f.Fields {
		if field.Name == name {
			return field
		}
	}
	return nil
}

// AddField adds a field to the form
func (f *Form) AddFields(field ...*Field) {
	if f.Fields == nil {
		f.Fields = make([]*Field, 0)
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
	var fields = make([]*Field, 0)
	for _, field := range f.Fields {
		var found = false
		for _, name := range names {
			if strings.EqualFold(field.Name, name) {
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
			field.Disabled = true
		}
		return *f
	}
	for _, field := range f.Fields {
		for _, name := range names {
			if strings.EqualFold(field.Name, name) {
				field.Disabled = true
				break
			}
		}
	}
	return *f
}

func (f *Form) Get(name string) string {
	for _, field := range f.Fields {
		if field.Name == name {
			return field.Value
		}
	}
	return ""
}
