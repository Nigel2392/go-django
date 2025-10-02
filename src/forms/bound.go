package forms

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"html/template"
	"io"
	"runtime/debug"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/logger"
)

type FormRenderer interface {
	Render(w io.Writer, ctx context.Context, field BoundField, id string, name string, value interface{}, errors []error, attrs map[string]string)
	RenderLabel(w io.Writer, ctx context.Context, field BoundField, id string, name string)
	RenderHelpText(w io.Writer, ctx context.Context, field BoundField, id string, name string)
	RenderField(w io.Writer, ctx context.Context, field BoundField, id string, name string, value interface{}, attrs map[string]string, errors []error)
}

type defaultRenderer struct{}

func (r *defaultRenderer) Render(w io.Writer, c context.Context, field BoundField, id string, name string, value interface{}, errors []error, attrs map[string]string, widgetCtx ctx.Context) {
	r.RenderLabel(w, c, field, id, name)
	r.RenderHelpText(w, c, field, id, name)
	r.RenderField(w, c, field, id, name, value, attrs, errors, widgetCtx)
}

func (r *defaultRenderer) RenderLabel(w io.Writer, ctx context.Context, field BoundField, id string, name string) {
	var fld = field.Input()
	var labelText = fld.Label(ctx)
	fmt.Fprintf(w, "<label for=\"%s\">%s</label>", id, html.EscapeString(labelText))
}

func (r *defaultRenderer) RenderHelpText(w io.Writer, ctx context.Context, field BoundField, id string, name string) {
	var fld = field.Input()
	var helpText = fld.HelpText(ctx)
	if helpText == "" {
		return
	}

	fmt.Fprintf(w, "<small>%s</small>", html.EscapeString(helpText))
}

func (r *defaultRenderer) RenderField(w io.Writer, c context.Context, field BoundField, id string, name string, value interface{}, attrs map[string]string, errors []error, widgetCtx ctx.Context) {
	var widget = field.Widget()
	var err = widget.RenderWithErrors(
		c, w, id, name, value, errors, attrs, widgetCtx,
	)
	assert.True(err == nil, err)
}

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

func NewBoundFormField(ctx context.Context, w Widget, f Field, name string, value interface{}, errors []error, tryWidgetBound bool) BoundField {

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
		defer func() {
			if r := recover(); r != nil {
				logger.Errorf("panic in template %q: %v\n%s", b.FormField.Name(), r, debug.Stack())
			}
		}()

		var widgetCtx = b.FormWidget.GetContextData(b.FormContext, b.FormID, b.FormName, b.FormValue, b.FormAttrs)
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

func (b *BoundFormField) Context() context.Context {
	return b.FormContext
}

func (b *BoundFormField) Errors() []error {
	return b.FormErrors
}
