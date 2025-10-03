package forms

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"html/template"
	"io"
	"runtime/debug"
	"strings"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/logger"
)

type FormRenderer interface {
	RenderAsP(w io.Writer, ctx context.Context, form BoundForm) error
	RenderAsUL(w io.Writer, ctx context.Context, form BoundForm) error
	RenderAsTable(w io.Writer, ctx context.Context, form BoundForm) error

	RenderFieldLabel(w io.Writer, ctx context.Context, field BoundField, id string, name string) error
	RenderFieldHelpText(w io.Writer, ctx context.Context, field BoundField, id string, name string) error
	RenderFieldWidget(w io.Writer, ctx context.Context, field BoundField, id string, name string, value interface{}, attrs map[string]string, errors []error, widgetCtx ctx.Context) error
	RenderField(w io.Writer, ctx context.Context, field BoundField, id string, name string, value interface{}, errors []error, attrs map[string]string, widgetCtx ctx.Context) error
}

type defaultRenderer struct{}

func (r *defaultRenderer) RenderAsP(w io.Writer, c context.Context, form BoundForm) error {
	for _, field := range form.Fields() {
		w.Write([]byte("<p>"))
		w.Write([]byte(field.Label()))
		w.Write([]byte(field.HelpText()))
		w.Write([]byte(field.Field()))
		w.Write([]byte("</p>"))
	}
	return nil
}

func (r *defaultRenderer) RenderAsUL(w io.Writer, c context.Context, form BoundForm) error {
	w.Write([]byte("<ul>"))
	for _, field := range form.Fields() {
		w.Write([]byte("<li>"))
		w.Write([]byte(field.Label()))
		w.Write([]byte("</li>"))

		w.Write([]byte("<li>"))
		w.Write([]byte(field.HelpText()))
		w.Write([]byte("</li>"))

		w.Write([]byte("<li>"))
		w.Write([]byte(field.Field()))
		w.Write([]byte("</li>"))

	}
	w.Write([]byte("</ul>"))
	return nil
}

func (r *defaultRenderer) RenderAsTable(w io.Writer, c context.Context, form BoundForm) error {
	w.Write([]byte("<table>"))
	for _, field := range form.Fields() {
		w.Write([]byte("<tr>"))
		w.Write([]byte("<td>"))
		w.Write([]byte(field.Label()))
		w.Write([]byte("</td>"))
		w.Write([]byte("<td>"))
		w.Write([]byte(field.HelpText()))
		w.Write([]byte(field.Field()))
		w.Write([]byte("</td>"))
		w.Write([]byte("</tr>"))
	}
	w.Write([]byte("</table>"))
	return nil
}

func (r *defaultRenderer) RenderField(w io.Writer, c context.Context, field BoundField, id string, name string, value interface{}, errors []error, attrs map[string]string, widgetCtx ctx.Context) (err error) {
	if err = r.RenderFieldLabel(w, c, field, id, name); err != nil {
		return err
	}
	if err = r.RenderFieldWidget(w, c, field, id, name, value, attrs, errors, widgetCtx); err != nil {
		return err
	}
	if err = r.RenderFieldHelpText(w, c, field, id, name); err != nil {
		return err
	}
	return nil
}

func (r *defaultRenderer) RenderFieldLabel(w io.Writer, ctx context.Context, field BoundField, id string, name string) error {
	var fld = field.Input()
	var labelText = fld.Label(ctx)
	fmt.Fprintf(w,
		"<label for=\"%s\">%s</label>",
		id, html.EscapeString(labelText),
	)
	return nil
}

func (r *defaultRenderer) RenderFieldHelpText(w io.Writer, ctx context.Context, field BoundField, id string, name string) error {
	var fld = field.Input()
	var helpText = fld.HelpText(ctx)
	if helpText == "" {
		return nil
	}

	w.Write([]byte(html.EscapeString(helpText)))
	return nil
}

func (r *defaultRenderer) RenderFieldWidget(w io.Writer, c context.Context, field BoundField, id string, name string, value interface{}, attrs map[string]string, errors []error, widgetCtx ctx.Context) error {
	return field.Widget().RenderWithErrors(
		c, w, id, name, value, errors, attrs, widgetCtx,
	)
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
	Renderer    FormRenderer
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
		Renderer:    &defaultRenderer{},
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
