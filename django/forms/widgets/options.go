package widgets

import (
	"io"

	"github.com/Nigel2392/django/core/ctx"
	"github.com/Nigel2392/django/core/tpl"
)

type Option interface {
	Label() string
	Value() string
}

type FormOption struct {
	OptLabel string
	OptValue string
}

func (o *FormOption) Label() string {
	return o.OptLabel
}

func (o *FormOption) Value() string {
	return o.OptValue
}

func NewOption(name, label, value string) Option {
	return &FormOption{OptLabel: label, OptValue: value}
}

type OptionsWidget struct {
	*BaseWidget
	Choices      func() []Option
	IncludeBlank bool
	BlankLabel   string
}

func NewOptionWidget(type_ func() string, templateName string, attrs map[string]string, choices func() []Option) Widget {
	return &OptionsWidget{
		BaseWidget: NewBaseWidget(type_, templateName, attrs),
		Choices:    choices,
	}
}

func NewCheckboxInput(attrs map[string]string, choices func() []Option) Widget {
	return NewOptionWidget(S("checkbox"), "forms/widgets/checkbox.html", attrs, choices)
}

func NewRadioInput(attrs map[string]string, choices func() []Option) Widget {
	return NewOptionWidget(S("radio"), "forms/widgets/radio.html", attrs, choices)
}

func NewSelectInput(attrs map[string]string, choices func() []Option) Widget {
	return NewOptionWidget(S("select"), "forms/widgets/select.html", attrs, choices)
}

func (o *OptionsWidget) GetContextData(id, name string, value interface{}, attrs map[string]string) ctx.Context {
	var base_context = o.BaseWidget.GetContextData(id, name, value, attrs)
	var choices = o.Choices()
	base_context.Set("choices", choices)
	base_context.Set("include_blank", o.IncludeBlank)
	if o.BlankLabel == "" {
		o.BlankLabel = "---------"
	}
	base_context.Set("blank_label", o.BlankLabel)
	return base_context
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
