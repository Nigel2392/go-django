package forms

import (
	"html/template"
	"strings"
)

type FormError struct {
	Name     string
	FieldErr error
}

func (f FormError) Error() string {
	var b strings.Builder
	b.WriteString(f.Name)
	b.WriteString(": ")
	b.WriteString(f.FieldErr.Error())
	return b.String()
}

type FormErrors []FormError

func (f *FormErrors) Add(name string, err error) {
	if *f == nil {
		*f = make(FormErrors, 0)
	}
	*f = append(*f, FormError{
		Name:     name,
		FieldErr: err,
	})
}

func (f FormErrors) AsP() template.HTML {
	var b strings.Builder
	for _, err := range f {
		b.WriteString("<p class=\"error\">")
		b.WriteString(err.Error())
		b.WriteString("</p>\r\n")
	}
	return template.HTML(b.String())
}

func (f FormErrors) AsUL() template.HTML {
	var b strings.Builder
	b.WriteString("<ul class=\"error\">\r\n")
	for _, err := range f {
		b.WriteString("<li>")
		b.WriteString(err.Error())
		b.WriteString("</li>\r\n")
	}
	b.WriteString("</ul>\r\n")
	return template.HTML(b.String())
}

func (f FormErrors) String() string {
	return f.Error()
}

func (f FormErrors) Error() string {
	var b strings.Builder
	for _, err := range f {
		b.WriteString(err.Error())
		b.WriteString("\r\n")
	}
	return b.String()
}

func (f FormErrors) HasErrors() bool {
	return len(f) > 0
}
