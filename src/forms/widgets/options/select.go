package options

import (
	"context"
	"io"
	"net/url"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/go-django/src/forms/widgets"
)

type OptionsWidget struct {
	*widgets.BaseWidget
	Choices      func() []widgets.Option
	IncludeBlank bool
	BlankLabel   string
}

func NewOptionWidget(type_ string, templateName string, attrs map[string]string, choices func() []widgets.Option, options ...func(*OptionsWidget)) *OptionsWidget {
	var w = &OptionsWidget{
		BaseWidget: widgets.NewBaseWidget(type_, templateName, attrs),
		Choices:    choices,
	}
	for _, opt := range options {
		opt(w)
	}
	return w
}

func NewCheckboxInput(attrs map[string]string, choices func() []widgets.Option, opts ...func(*OptionsWidget)) *OptionsWidget {
	return NewOptionWidget("checkbox", "forms/widgets/checkbox.html", attrs, choices, opts...)
}

func NewRadioInput(attrs map[string]string, choices func() []widgets.Option, opts ...func(*OptionsWidget)) *OptionsWidget {
	return NewOptionWidget("radio", "forms/widgets/radio.html", attrs, choices, opts...)
}

func NewSelectInput(attrs map[string]string, choices func() []widgets.Option, opts ...func(*OptionsWidget)) *OptionsWidget {
	return NewOptionWidget("select", "forms/widgets/select.html", attrs, choices, opts...)
}

func (o *OptionsWidget) GetContextData(ctx context.Context, id, name string, value interface{}, attrs map[string]string) ctx.Context {
	var base_context = o.BaseWidget.GetContextData(ctx, id, name, value, attrs)
	var choices = o.Choices()

	var values []string
	if value != nil {
		switch v := value.(type) {
		case string:
			values = []string{v}
		case []string:
			values = v
		}
	}

	base_context.Set(
		"choices",
		widgets.WrapOptions(choices, values),
	)

	base_context.Set("include_blank", o.IncludeBlank)
	if o.BlankLabel == "" {
		o.BlankLabel = "---------"
	}
	base_context.Set("blank_label", o.BlankLabel)
	return base_context
}

func (o *OptionsWidget) Validate(ctx context.Context, value interface{}) []error {
	if value == nil {
		return nil
	}

	var (
		errors  []error
		choices = o.Choices()
		values  []string
	)

	switch v := value.(type) {
	case string:
		values = []string{v}
	case []string:
		values = v
	}

	if (len(values) == 0 || len(values) == 1 && values[0] == "") && o.IncludeBlank {
		return nil
	}

	for _, valueStr := range values {
		var found bool
		for _, choice := range choices {
			if choice.Value() == valueStr {
				found = true
				break
			}
		}

		if !found {
			errors = append(errors, errs.ErrInvalidValue)
		}
	}

	return errors
}

func (b *OptionsWidget) Render(ctx context.Context, w io.Writer, id, name string, value interface{}, attrs map[string]string) error {
	return b.RenderWithErrors(ctx, w, id, name, value, nil, attrs, b.GetContextData(ctx, id, name, value, attrs))
}

type MultiSelectWidget struct {
	*OptionsWidget
}

func NewMultiSelectInput(attrs map[string]string, choices func() []widgets.Option) widgets.Widget {
	return &MultiSelectWidget{
		OptionsWidget: NewOptionWidget("select", "forms/widgets/multi-select.html", attrs, choices),
	}
}

func (m *MultiSelectWidget) ValueFromDataDict(ctx context.Context, data url.Values, files map[string][]filesystem.FileHeader, name string) (interface{}, []error) {
	var values, ok = data[name]
	if !ok {
		return nil, nil
	}
	return values, nil
}

func (m *MultiSelectWidget) Media() media.Media {
	var formMedia = media.NewMedia()
	formMedia.AddCSS(media.CSS(django.Static(
		"forms/css/multiple-select.css",
	)))
	formMedia.AddJS(&media.JSAsset{
		URL: django.Static(
			"forms/js/index.js",
		),
	})
	return formMedia
}

func IncludeBlank(b bool) func(*OptionsWidget) {
	return func(o *OptionsWidget) {
		o.IncludeBlank = b
	}
}

func BlankLabel(label string) func(*OptionsWidget) {
	return func(o *OptionsWidget) {
		o.BlankLabel = label
	}
}
