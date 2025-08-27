package forms

import (
	"bytes"
	"context"
	"fmt"
	"html/template"

	"github.com/Nigel2392/go-django/src/core/assert"
)

type BoundFormField struct {
	FormWidget  Widget
	FormField   Field
	FormName    string
	FormID      string
	FormAttrs   map[string]string
	FormValue   interface{}
	FormErrors  []error
	FormContext context.Context
	CachedHTML  template.HTML
}

func NewBoundFormField(ctx context.Context, w Widget, f Field, name string, value interface{}, errors []error) BoundField {

	if errors == nil {
		errors = make([]error, 0)
	}

	var attrs = f.Attrs()
	if attrs == nil {
		attrs = make(map[string]string)
	}

	var bw = &BoundFormField{
		FormWidget: w,
		FormField:  f,
		FormName:   name,
		FormID: fmt.Sprintf(
			"id_%s", w.IdForLabel(name),
		),
		FormValue:   value,
		FormErrors:  errors,
		FormAttrs:   attrs,
		FormContext: ctx,
	}

	return bw
}

func (b *BoundFormField) Widget() Widget {
	return b.FormWidget
}

func (b *BoundFormField) Input() Field {
	return b.FormField
}

func (b *BoundFormField) ID() string {
	return b.FormID
}

func (b *BoundFormField) Hidden() bool {
	return b.FormWidget.IsHidden()
}

func (b *BoundFormField) Label() template.HTML {
	var (
		labelText = b.FormField.Label(b.FormContext)
	)
	return template.HTML(
		fmt.Sprintf("<label for=\"%s\">%s</label>", b.ID(), labelText),
	)
}

func (b *BoundFormField) HelpText() template.HTML {
	var (
		helpText = b.FormField.HelpText(b.FormContext)
	)
	return template.HTML(helpText)
}

func (b *BoundFormField) Field() template.HTML {
	if b.FormContext == nil {
		panic("BoundFormField: FormContext is nil")
	}

	if b.CachedHTML == "" {
		var attrs = b.FormField.Attrs()
		if attrs == nil {
			attrs = make(map[string]string)
		}

		var widgetCtx = b.FormWidget.GetContextData(b.FormContext, b.FormID, b.FormName, b.FormValue, attrs)
		if len(b.FormErrors) > 0 {
			widgetCtx.Set("errors", b.FormErrors)
		}

		var err error
		var buf = new(bytes.Buffer)
		err = b.FormWidget.RenderWithErrors(
			b.FormContext, buf, b.FormID, b.FormName, b.FormValue, b.FormErrors, b.FormAttrs, widgetCtx,
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
