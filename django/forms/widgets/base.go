package widgets

import (
	"html/template"
	"io"
	"maps"
	"net/url"

	"github.com/Nigel2392/django/core/ctx"
	"github.com/Nigel2392/django/core/tpl"
	"github.com/Nigel2392/django/forms/media"
)

type BaseWidget struct {
	Type          func() string
	TemplateName  string
	InputIsHidden bool
	BaseAttrs     map[string]string
}

func (b *BaseWidget) Hide(hidden bool) {
	b.InputIsHidden = hidden
}

func (b *BaseWidget) SetAttrs(attrs map[string]string) {
	if b.BaseAttrs == nil {
		b.BaseAttrs = make(map[string]string)
	}
	maps.Copy(b.BaseAttrs, attrs)
}

func (b *BaseWidget) IsHidden() bool {
	return b.InputIsHidden
}

func (b *BaseWidget) IdForLabel(id string) string {
	return id
}

func (b *BaseWidget) ValueToGo(value interface{}) (interface{}, error) {
	return value, nil
}

func (b *BaseWidget) ValueToForm(value interface{}) interface{} {
	return value
}

func (b *BaseWidget) ValueOmittedFromData(data url.Values, files map[string][]io.ReadCloser, name string) bool {
	return !data.Has(name)
}

func (b *BaseWidget) ValueFromDataDict(data url.Values, files map[string][]io.ReadCloser, name string) (interface{}, []error) {
	var value string
	if data.Has(name) {
		value = data.Get(name)
	}
	return value, nil
}

func (b *BaseWidget) GetContextData(id, name string, value interface{}, attrs map[string]string) ctx.Context {
	var widgetAttrs = make(map[string]string)
	maps.Copy(widgetAttrs, b.BaseAttrs)
	maps.Copy(widgetAttrs, attrs)

	var type_ = b.Type()
	if widgetAttrs["type"] != "" {
		type_ = widgetAttrs["type"]
		delete(widgetAttrs, "type")
	}

	if b.InputIsHidden {
		type_ = "hidden"
	}

	return ctx.NewContext(map[string]interface{}{
		"id":     id,
		"type":   type_,
		"name":   name,
		"value":  value,
		"attrs":  attrs,
		"widget": b,
	})
}

func (b *BaseWidget) RenderWithErrors(id, name string, value interface{}, errors []error, attrs map[string]string) (template.HTML, error) {
	var context = b.GetContextData(id, name, value, attrs)
	if errors != nil {
		context.Set("errors", errors)
	}

	return tpl.Render(context, b.TemplateName)
}

func (b *BaseWidget) Render(id, name string, value interface{}, attrs map[string]string) (template.HTML, error) {
	return b.RenderWithErrors(id, name, value, nil, attrs)
}

func (b *BaseWidget) Media() media.Media {
	return nil
}

func NewBaseWidget(type_ func() string, templateName string, attrs map[string]string) *BaseWidget {

	if attrs == nil {
		attrs = make(map[string]string)
	}

	if templateName == "" {
		templateName = "forms/widgets/input.html"
	}

	if type_ == nil {
		type_ = S("text")
	}

	return &BaseWidget{
		Type:          type_,
		TemplateName:  templateName,
		InputIsHidden: false,
		BaseAttrs:     attrs,
	}
}
