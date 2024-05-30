package forms

import (
	"fmt"
	"html/template"

	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/django/forms/widgets"
)

type BoundFormWidget struct {
	FormWidget widgets.Widget
	FormField  fields.Field
	FormName   string
	FormAttrs  map[string]string
	FormValue  interface{}
	FormErrors []error
	CachedHTML template.HTML
}

func NewBoundFormWidget(w widgets.Widget, f fields.Field, name string, value interface{}, errors []error) BoundField {

	if errors == nil {
		errors = make([]error, 0)
	}

	var bw = &BoundFormWidget{
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

func (b *BoundFormWidget) Widget() widgets.Widget {
	return b.FormWidget
}

func (b *BoundFormWidget) Input() fields.Field {
	return b.FormField
}

func (b *BoundFormWidget) ID() string {
	return fmt.Sprintf(
		"id_%s", b.FormWidget.IdForLabel(b.FormName),
	)
}

func (b *BoundFormWidget) Label() template.HTML {
	var (
		labelText = b.FormField.Label()
	)
	return template.HTML(
		fmt.Sprintf("<label for=\"%s\">%s</label>", b.ID(), labelText),
	)
}

func (b *BoundFormWidget) Field() template.HTML {
	if b.CachedHTML == "" {
		var err error
		b.CachedHTML, err = b.FormWidget.RenderWithErrors(
			b.ID(), b.FormName, b.FormValue, b.FormErrors, b.FormAttrs,
		)
		assert.True(err == nil, err)
	}
	return b.CachedHTML
}

func (b *BoundFormWidget) HTML() template.HTML {
	return template.HTML(
		fmt.Sprintf("%s%s", b.Label(), b.Field()),
	)
}

func (b *BoundFormWidget) Name() string {
	return b.FormName
}

func (b *BoundFormWidget) Attrs() map[string]string {
	return b.FormAttrs
}

func (b *BoundFormWidget) Value() interface{} {
	return b.FormValue
}

func (b *BoundFormWidget) Errors() []error {
	return b.FormErrors
}
