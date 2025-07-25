package chooser

import (
	"context"
	"fmt"
	"io"

	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/forms/widgets"
)

var _ widgets.Widget = &Select{}

type Select struct {
	*BaseChooser
	ExcludeBlank bool
	BlankLabel   string
}

func SelectWidget(allowBlank bool, blankLabel string, opts BaseChooserOptions, attrs map[string]string) *Select {
	var chooser = BaseChooserWidget(opts, attrs)
	chooser.BaseWidget.Type = "select"
	chooser.BaseWidget.TemplateName = "forms/widgets/select.html"
	return &Select{
		BaseChooser:  chooser,
		ExcludeBlank: !allowBlank,
		BlankLabel:   blankLabel,
	}
}

func (o *Select) ValueToForm(value interface{}) interface{} {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%v", value)
}

func (o *Select) GetContextData(ctx context.Context, id, name string, value interface{}, widgetAttrs map[string]string) ctx.Context {
	var base_context = o.BaseWidget.GetContextData(ctx, id, name, value, widgetAttrs)
	var modelInstances, err = o.QuerySet(ctx)
	if err != nil {
		logger.Errorf(
			"error getting model instances for model: %s, %s",
			o.forModelDefinition.Name(),
			err,
		)
		return base_context
	}

	var choices = make([]widgets.Option, 0, len(modelInstances))
	for _, modelInstance := range modelInstances {

		var value = o.Opts.GetPrimaryKey(modelInstance)
		var labelStr = o.forModelDefinition.InstanceLabel(
			modelInstance,
		)
		var valueStr = fmt.Sprintf("%v", value)
		var option = widgets.NewOption(
			valueStr, labelStr, valueStr,
		)

		choices = append(choices, option)
	}

	var values []string
	if value != nil {
		switch v := value.(type) {
		case string:
			values = []string{v}
		case []string:
			values = v
		default:
			values = []string{fmt.Sprintf("%v", v)}
		}
	}

	base_context.Set(
		"choices",
		widgets.WrapOptions(choices, values),
	)

	base_context.Set("include_blank", !o.ExcludeBlank)
	if o.BlankLabel == "" {
		o.BlankLabel = "---------"
	}
	base_context.Set("blank_label", o.BlankLabel)
	return base_context
}

func (b *Select) RenderWithErrors(ctx context.Context, w io.Writer, id, name string, value interface{}, errors []error, attrs map[string]string) error {
	var context = b.GetContextData(ctx, id, name, value, attrs)
	if errors != nil {
		context.Set("errors", errors)
	}

	return tpl.FRender(w, context, b.TemplateName)
}

func (b *Select) Render(ctx context.Context, w io.Writer, id, name string, value interface{}, attrs map[string]string) error {
	return b.RenderWithErrors(ctx, w, id, name, value, nil, attrs)
}
