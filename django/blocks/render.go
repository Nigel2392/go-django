package blocks

import (
	"html/template"

	"github.com/Nigel2392/django/core/ctx"
	"github.com/Nigel2392/django/core/errs"
	"github.com/Nigel2392/django/core/tpl"
)

func RenderBlockForm(widget *BlockWidget, context *BlockContext) (template.HTML, error) {
	var (
		template = "blocks/forms/widgets/block_widget.tmpl"
	)

	var b, err = widget.BlockDef.RenderForm(context.ID, context.Name, context.Value, context)
	if err != nil {
		return "", errs.Error(b)
	}

	context.BlockHTML = b

	return tpl.Render(context, template)
}

func RenderBlock(block Block, value any, context ctx.Context) (template.HTML, error) {
	return block.Render(value, context)
}
