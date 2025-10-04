package blocks

import (
	"bytes"
	"context"
	"io"

	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/Nigel2392/go-telepath/telepath"
)

type FieldBlock struct {
	*BaseBlock
}

func NewFieldBlock(opts ...func(*FieldBlock)) *FieldBlock {
	var base = &FieldBlock{
		BaseBlock: NewBaseBlock(),
	}
	runOpts(opts, base)
	return base
}

func (b *FieldBlock) RenderForm(ctx context.Context, w io.Writer, id, name string, value interface{}, errors []error, c ctx.Context) error {
	var blockArgs = map[string]interface{}{
		"id":     id,
		"name":   name,
		"value":  value,
		"errors": errors,
		"type":   b.Field().Widget().FormType(),
		"block":  b,
	}
	var bt, err = telepath.PackJSON(ctx, JSContext, blockArgs)
	if err != nil {
		return err
	}

	return b.RenderTempl(id, name, value, string(bt), errors, c).Render(ctx, w)
}

func (b *FieldBlock) RenderHTML(ctx context.Context) string {
	var w = new(bytes.Buffer)
	var widget = b.FormField.Widget()
	widget.Render(
		ctx,
		w,
		widget.IdForLabel("__PREFIX__"),
		"__PREFIX__",
		b.GetDefault(),
		b.FormField.Attrs(),
	)
	return w.String()
}

func CharBlock(opts ...func(*FieldBlock)) *FieldBlock {
	var base = NewFieldBlock(opts...)
	base.Template = "blocks/templates/text.html"
	base.SetField(fields.CharField())
	return base
}

func NumberBlock(opts ...func(*FieldBlock)) *FieldBlock {
	var base = NewFieldBlock(opts...)
	base.Template = "blocks/templates/number.html"
	base.SetField(fields.NumberField[int]())
	return base
}

func TextBlock(opts ...func(*FieldBlock)) *FieldBlock {
	var base = NewFieldBlock(opts...)
	base.Template = "blocks/templates/text.html"
	base.SetField(fields.CharField(
		fields.Widget(widgets.NewTextarea(nil)),
	))
	return base
}

func EmailBlock(opts ...func(*FieldBlock)) *FieldBlock {
	var base = NewFieldBlock(opts...)
	base.Template = "blocks/templates/email.html"
	base.SetField(fields.EmailField())
	return base
}

func PasswordBlock(opts ...func(*FieldBlock)) *FieldBlock {
	var base = NewFieldBlock(opts...)
	base.Template = "blocks/templates/password.html"
	base.SetField(fields.CharField(
		fields.Widget(widgets.NewPasswordInput(nil)),
	))
	return base
}

func DateBlock(opts ...func(*FieldBlock)) *FieldBlock {
	var base = NewFieldBlock(opts...)
	base.Template = "blocks/templates/date.html"
	base.SetField(fields.DateField(widgets.DateWidgetTypeDate))
	// base.Default = func() interface{} {
	// return time.Time{}
	// }
	return base
}

func DateTimeBlock(opts ...func(*FieldBlock)) *FieldBlock {
	var base = NewFieldBlock(opts...)
	base.Template = "blocks/templates/datetime.html"
	base.SetField(fields.DateField(widgets.DateWidgetTypeDateTime))
	// base.Default = func() interface{} {
	// return time.Time{}
	// }
	return base
}
