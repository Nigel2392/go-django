package forms

import (
	"bytes"
	"context"
	"html/template"
	"runtime/debug"
	"strings"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/logger"
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
	Renderer    FormRenderer
}

func NewBoundFormField(ctx context.Context, renderer FormRenderer, w Widget, f Field, name string, value interface{}, errors []error, tryWidgetBound bool) BoundField {

	if tryWidgetBound {
		if bw, ok := w.(BinderWidget); ok {
			return bw.BoundField(ctx, w, f, name, value, errors)
		}
	}

	if errors == nil {
		errors = make([]error, 0)
	}

	var attrs = f.Attrs()
	if attrs == nil {
		attrs = make(map[string]string)
	}

	// Bind the field to the widget
	w.BindField(f)

	if renderer == nil {
		renderer = &defaultRenderer{}
	}

	var bw = &BoundFormField{
		FormWidget:  w,
		FormField:   f,
		FormName:    name,
		FormID:      w.IdForLabel(name),
		FormValue:   value,
		FormErrors:  errors,
		FormAttrs:   attrs,
		FormContext: ctx,
		Renderer:    renderer,
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
	var s = new(strings.Builder)
	b.Renderer.RenderFieldLabel(s, b.FormContext, b, b.FormID, b.FormName)
	return template.HTML(s.String())
}

func (b *BoundFormField) HelpText() template.HTML {
	var s = new(strings.Builder)
	b.Renderer.RenderFieldHelpText(s, b.FormContext, b, b.FormID, b.FormName)
	return template.HTML(s.String())
}

func (b *BoundFormField) Field() template.HTML {
	if b.FormContext == nil {
		panic("BoundFormField: FormContext is nil")
	}

	if b.CachedHTML == "" {
		defer func() {
			if r := recover(); r != nil {
				logger.Errorf("panic in template %q: %v\n%s", b.FormField.Name(), r, debug.Stack())
			}
		}()

		var err error
		var buf = new(bytes.Buffer)
		var widgetCtx = b.getWidgetContext()
		err = b.Renderer.RenderFieldWidget(buf, b.FormContext, b, b.FormID, b.FormName, b.FormValue, b.FormAttrs, b.FormErrors, widgetCtx)
		b.CachedHTML = template.HTML(buf.String())
		assert.True(err == nil, err)
	}
	return b.CachedHTML
}

func (b *BoundFormField) getWidgetContext() ctx.Context {
	var widgetCtx = b.FormWidget.GetContextData(b.FormContext, b.FormID, b.FormName, b.FormValue, b.FormAttrs)
	if len(b.FormErrors) > 0 {
		widgetCtx.Set("errors", b.FormErrors)
	}
	return widgetCtx
}

func (b *BoundFormField) HTML() template.HTML {
	var widgetCtx = b.getWidgetContext()
	var buf = new(bytes.Buffer)
	var err = b.Renderer.RenderField(
		buf,
		b.FormContext,
		b,
		b.FormID,
		b.FormName,
		b.FormValue,
		b.FormErrors,
		b.FormAttrs,
		widgetCtx,
	)

	assert.True(err == nil, err)
	return template.HTML(buf.String())
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

func (b *BoundFormField) Context() context.Context {
	return b.FormContext
}

func (b *BoundFormField) Errors() []error {
	return b.FormErrors
}
