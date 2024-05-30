package widgets

import "github.com/Nigel2392/django/core/ctx"

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
	Choices func() []Option
}

func (o *OptionsWidget) GetContextData(id, name string, value interface{}, attrs map[string]string) ctx.Context {
	var base_context = o.BaseWidget.GetContextData(id, name, value, attrs)
	var choices = o.Choices()
	base_context.Set("choices", choices)
	return base_context
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
