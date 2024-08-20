package forms

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/django/forms/widgets"
)

type BoundFormField struct {
	FormWidget widgets.Widget
	FormField  fields.Field
	FormName   string
	FormAttrs  map[string]string
	FormValue  interface{}
	FormErrors []error
	CachedHTML template.HTML
}

func NewBoundFormField(w widgets.Widget, f fields.Field, name string, value interface{}, errors []error) BoundField {

	if errors == nil {
		errors = make([]error, 0)
	}

	var bw = &BoundFormField{
		FormWidget: w,
		FormField:  f,
		FormName:   name,
		FormValue:  value,
		FormErrors: errors,
	}

	var attrs = f.Attrs()
	if attrs != nil {
		bw.FormAttrs = attrs
	} else {
		bw.FormAttrs = make(map[string]string)
	}

	return bw
}

func (b *BoundFormField) Widget() widgets.Widget {
	return b.FormWidget
}

func (b *BoundFormField) Input() fields.Field {
	return b.FormField
}

func (b *BoundFormField) ID() string {
	return fmt.Sprintf(
		"id_%s", b.FormWidget.IdForLabel(b.FormName),
	)
}

func (b *BoundFormField) Label() template.HTML {
	var (
		labelText = b.FormField.Label()
	)
	return template.HTML(
		fmt.Sprintf("<label for=\"%s\">%s</label>", b.ID(), labelText),
	)
}

func (b *BoundFormField) HelpText() template.HTML {
	var (
		helpText = b.FormField.HelpText()
	)
	return template.HTML(helpText)
}

func (b *BoundFormField) Field() template.HTML {
	if b.CachedHTML == "" {
		var err error
		var buf = new(bytes.Buffer)
		err = b.FormWidget.RenderWithErrors(
			buf, b.ID(), b.FormName, b.FormValue, b.FormErrors, b.FormAttrs,
		)
		b.CachedHTML = template.HTML(buf.String())
		assert.True(err == nil, err)
	}
	return b.CachedHTML
}

func (b *BoundFormField) HTML() template.HTML {
	return template.HTML(
		fmt.Sprintf("%s%s", b.Label(), b.Field()),
	)
}

func (b *BoundFormField) Name() string {
	return b.FormName
}

func (b *BoundFormField) Attrs() map[string]string {
	return b.FormAttrs
}

func (b *BoundFormField) Value() interface{} {
	return b.FormValue
}

func (b *BoundFormField) Errors() []error {
	return b.FormErrors
}
