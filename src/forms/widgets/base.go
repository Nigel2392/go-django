package widgets

import (
	"context"
	"fmt"
	"io"
	"maps"
	"net/url"
	"strings"

	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/forms/media"
)

type BaseWidget struct {
	Type          string
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
	if value == nil {
		return nil
	}

	return fmt.Sprintf("%v", value)
}

func (b *BaseWidget) Validate(ctx context.Context, value interface{}) []error {
	return nil
}

func (b *BaseWidget) ValueOmittedFromData(ctx context.Context, data url.Values, files map[string][]filesystem.FileHeader, name string) bool {
	return !data.Has(name)
}

func (b *BaseWidget) ValueFromDataDict(ctx context.Context, data url.Values, files map[string][]filesystem.FileHeader, name string) (interface{}, []error) {
	var value string
	if data.Has(name) {
		value = strings.TrimSpace(data.Get(name))
	}
	return value, nil
}

func (b *BaseWidget) GetContextData(c context.Context, id, name string, value interface{}, attrs map[string]string) ctx.Context {
	var widgetAttrs = make(map[string]string)
	for k, v := range b.BaseAttrs {
		var attr, ok = attrs[k]
		if !ok {
			widgetAttrs[k] = v
		} else {
			widgetAttrs[k] = fmt.Sprintf("%s %s", v, attr)
		}
	}
	for k, v := range attrs {
		if _, ok := widgetAttrs[k]; !ok {
			widgetAttrs[k] = v
		}
	}

	var type_ = b.Type
	if widgetAttrs["type"] != "" {
		type_ = widgetAttrs["type"]
		delete(widgetAttrs, "type")
	}

	if b.InputIsHidden {
		type_ = "hidden"
	}

	return ctx.NewContext(map[string]interface{}{
		"id":      id,
		"type":    type_,
		"name":    name,
		"value":   value,
		"attrs":   widgetAttrs,
		"widget":  b,
		"context": c,
	})
}

func (b *BaseWidget) RenderWithErrors(ctx context.Context, w io.Writer, id, name string, value interface{}, errors []error, attrs map[string]string) error {
	var context = b.GetContextData(ctx, id, name, value, attrs)
	if errors != nil {
		context.Set("errors", errors)
	}

	return tpl.FRender(w, context, b.TemplateName)
}

func (b *BaseWidget) Render(ctx context.Context, w io.Writer, id, name string, value interface{}, attrs map[string]string) error {
	return b.RenderWithErrors(ctx, w, id, name, value, nil, attrs)
}

func (b *BaseWidget) FormType() string {
	return b.Type
}

func (b *BaseWidget) Media() media.Media {
	return media.NewMedia()
}

func NewBaseWidget(type_ string, templateName string, attrs map[string]string) *BaseWidget {

	if attrs == nil {
		attrs = make(map[string]string)
	}

	if templateName == "" {
		templateName = "forms/widgets/input.html"
	}

	if type_ == "" {
		type_ = "text"
	}

	return &BaseWidget{
		Type:          type_,
		TemplateName:  templateName,
		InputIsHidden: false,
		BaseAttrs:     attrs,
	}
}
