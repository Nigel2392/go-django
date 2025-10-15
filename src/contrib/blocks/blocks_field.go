package blocks

import (
	"context"
	"encoding/json"
	"io"
	"net/mail"
	"time"

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
	var bt, err = telepath.PackJSON(ctx, JSContext, b)
	if err != nil {
		return err
	}

	return b.RenderTempl(id, name, value, string(bt), errors, c).Render(ctx, w)
}

func CharBlock(opts ...func(*FieldBlock)) *FieldBlock {
	var base = NewFieldBlock(opts...)
	base.DataType = string("")
	base.Template = "blocks/templates/text.html"
	base.SetField(fields.CharField())
	return base
}

func NumberBlock(opts ...func(*FieldBlock)) *FieldBlock {
	var base = NewFieldBlock(opts...)
	base.DataType = int(0)
	base.Template = "blocks/templates/number.html"
	base.SetField(fields.NumberField[int]())
	return base
}

func TextBlock(opts ...func(*FieldBlock)) *FieldBlock {
	var base = NewFieldBlock(opts...)
	base.DataType = string("")
	base.Template = "blocks/templates/text.html"
	base.SetField(fields.CharField(
		fields.Widget(widgets.NewTextarea(nil)),
	))
	return base
}

func EmailBlock(opts ...func(*FieldBlock)) *FieldBlock {
	var base = NewFieldBlock(opts...)
	base.DataType = &mail.Address{}
	base.Template = "blocks/templates/email.html"
	base.SetField(fields.EmailField())
	return base
}

func PasswordBlock(opts ...func(*FieldBlock)) *FieldBlock {
	var base = NewFieldBlock(opts...)
	base.DataType = string("")
	base.Template = "blocks/templates/password.html"
	base.SetField(fields.CharField(
		fields.Widget(widgets.NewPasswordInput(nil)),
	))
	return base
}

var timeFmts = []string{
	"2006-01-02",
	"2006-01-02T15:04:05",
	"2006-01-02T15:04",
	time.RFC3339,
	time.RFC1123,
	time.RFC1123Z,
	time.RFC822,
	time.RFC822Z,
}

func timeTimeBlock(template string, fmt widgets.DateWidgetType, opts ...func(*FieldBlock)) *FieldBlock {
	var base = NewFieldBlock(opts...)
	base.DataType = time.Time{}
	base.Template = template
	base.SetField(fields.DateField(fmt))
	base.ValueFromDBFunc = func(b *BaseBlock, j json.RawMessage) (interface{}, error) {
		if len(j) == 0 {
			return nil, nil
		}

		var s string
		if err := json.Unmarshal(j, &s); err != nil {
			return nil, err
		}

		var varr []error
		for _, f := range timeFmts {
			if t, err := time.Parse(f, s); err == nil {
				return t, nil
			} else {
				varr = append(varr, err)
			}
		}

		if len(varr) > 0 {
			return nil, varr[0]
		}
		return nil, nil
	}
	return base
}

func DateBlock(opts ...func(*FieldBlock)) *FieldBlock {
	return timeTimeBlock("blocks/templates/date.html", widgets.DateWidgetTypeDate, opts...)
}

func DateTimeBlock(opts ...func(*FieldBlock)) *FieldBlock {
	return timeTimeBlock("blocks/templates/datetime.html", widgets.DateWidgetTypeDateTime, opts...)
}
