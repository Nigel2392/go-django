package options

import (
	"io"
	"net/url"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/go-django/src/forms/widgets"
)

type OptionsWidget struct {
	*widgets.BaseWidget
	Choices      func() []widgets.Option
	IncludeBlank bool
	BlankLabel   string
}

func NewOptionWidget(type_ string, templateName string, attrs map[string]string, choices func() []widgets.Option) *OptionsWidget {
	return &OptionsWidget{
		BaseWidget: widgets.NewBaseWidget(type_, templateName, attrs),
		Choices:    choices,
	}
}

func NewCheckboxInput(attrs map[string]string, choices func() []widgets.Option) *OptionsWidget {
	return NewOptionWidget("checkbox", "forms/widgets/checkbox.html", attrs, choices)
}

func NewRadioInput(attrs map[string]string, choices func() []widgets.Option) *OptionsWidget {
	return NewOptionWidget("radio", "forms/widgets/radio.html", attrs, choices)
}

func NewSelectInput(attrs map[string]string, choices func() []widgets.Option) *OptionsWidget {
	return NewOptionWidget("select", "forms/widgets/select.html", attrs, choices)
}

func (o *OptionsWidget) GetContextData(id, name string, value interface{}, attrs map[string]string) ctx.Context {
	var base_context = o.BaseWidget.GetContextData(id, name, value, attrs)
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

func (o *OptionsWidget) Validate(value interface{}) []error {
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

func (b *OptionsWidget) RenderWithErrors(w io.Writer, id, name string, value interface{}, errors []error, attrs map[string]string) error {
	var context = b.GetContextData(id, name, value, attrs)
	if errors != nil {
		context.Set("errors", errors)
	}

	return tpl.FRender(w, context, b.TemplateName)
}

func (b *OptionsWidget) Render(w io.Writer, id, name string, value interface{}, attrs map[string]string) error {
	return b.RenderWithErrors(w, id, name, value, nil, attrs)
}

type MultiSelectWidget struct {
	*OptionsWidget
}

func NewMultiSelectInput(attrs map[string]string, choices func() []widgets.Option) widgets.Widget {
	return &MultiSelectWidget{
		OptionsWidget: NewOptionWidget("select", "forms/widgets/multi-select.html", attrs, choices),
	}
}

func (m *MultiSelectWidget) ValueFromDataDict(data url.Values, files map[string][]filesystem.FileHeader, name string) (interface{}, []error) {
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
			"forms/js/multiple-select.js",
		),
	})
	return formMedia
}
