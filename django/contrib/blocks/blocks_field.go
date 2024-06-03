package blocks

import (
	"context"
	"io"

	"github.com/Nigel2392/django/core/ctx"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/django/forms/widgets"
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

func (b *FieldBlock) RenderForm(w io.Writer, id, name string, value interface{}, errors []error, c ctx.Context) error {

	var blockArgs = map[string]interface{}{
		"id":    id,
		"name":  name,
		"value": value,
	}

	return b.RenderTempl(w, id, name, value, blockArgs, errors, c).Render(context.Background(), w)
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
