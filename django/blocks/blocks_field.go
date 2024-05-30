package blocks

import (
	"bytes"
	"html/template"

	"github.com/Nigel2392/django/core/ctx"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/django/forms/widgets"
)

type FieldBlock struct {
	*BaseBlock
}

func (b *FieldBlock) RenderForm(id, name string, value interface{}, errors []error, context ctx.Context) (template.HTML, error) {
	var c = context.(*BlockContext)
	var buf = new(bytes.Buffer)
	var html, err = b.Field().Widget().RenderWithErrors(id, name, value, errors, c.Attrs)
	if err != nil {
		return "", err
	}

	var label = b.Field().Label()
	var idForLabel = b.Field().Widget().IdForLabel(id)
	if len(errors) > 0 {
		buf.WriteString("<ul class=\"errorlist\">")

		for _, err := range errors {
			if err == nil {
				continue
			}
			buf.WriteString("<li>")
			buf.WriteString(err.Error())
			buf.WriteString("</li>")
		}

		buf.WriteString("</ul>")
	}

	buf.WriteString("<div class=\"field\">")
	buf.WriteString("<label for=\"")
	buf.WriteString(idForLabel)
	buf.WriteString("\">")
	buf.WriteString(label)
	buf.WriteString("</label>")
	buf.WriteString(string(html))
	buf.WriteString("</div>")
	return template.HTML(buf.String()), nil

}

func NewFieldBlock(opts ...func(*FieldBlock)) *FieldBlock {
	var base = &FieldBlock{
		BaseBlock: NewBaseBlock(),
	}
	runOpts(opts, base)
	return base
}

func CharBlock(opts ...func(*FieldBlock)) Block {
	var base = NewFieldBlock(opts...)
	base.Template = "blocks/templates/text.html"
	base.SetField(fields.CharField())
	return base
}

func NumberBlock(opts ...func(*FieldBlock)) Block {
	var base = NewFieldBlock(opts...)
	base.Template = "blocks/templates/number.html"
	base.SetField(fields.NumberField[int]())
	return base
}

func TextBlock(opts ...func(*FieldBlock)) Block {
	var base = NewFieldBlock(opts...)
	base.Template = "blocks/templates/text.html"
	base.SetField(fields.CharField(
		fields.Widget(widgets.NewTextarea(nil)),
	))
	return base
}

func EmailBlock(opts ...func(*FieldBlock)) Block {
	var base = NewFieldBlock(opts...)
	base.Template = "blocks/templates/email.html"
	base.SetField(fields.EmailField())
	return base
}

func PasswordBlock(opts ...func(*FieldBlock)) Block {
	var base = NewFieldBlock(opts...)
	base.Template = "blocks/templates/password.html"
	base.SetField(fields.CharField(
		fields.Widget(widgets.NewPasswordInput(nil)),
	))
	return base
}

func DateBlock(opts ...func(*FieldBlock)) Block {
	var base = NewFieldBlock(opts...)
	base.Template = "blocks/templates/date.html"
	base.SetField(fields.DateField(widgets.DateWidgetTypeDate))
	// base.Default = func() interface{} {
	// return time.Time{}
	// }
	return base
}

func DateTimeBlock(opts ...func(*FieldBlock)) Block {
	var base = NewFieldBlock(opts...)
	base.Template = "blocks/templates/datetime.html"
	base.SetField(fields.DateField(widgets.DateWidgetTypeDateTime))
	// base.Default = func() interface{} {
	// return time.Time{}
	// }
	return base
}
