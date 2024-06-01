package blocks

import (
	"bytes"
	"html/template"
	"io"

	"github.com/Nigel2392/django/core/ctx"
	"github.com/Nigel2392/django/core/errs"
	"github.com/Nigel2392/django/core/tpl"
)

func RenderBlockForm(w io.Writer, widget *BlockWidget, context *BlockContext, errors []error) error {
	var (
		byteBuf      = new(bytes.Buffer)
		templateName = "blocks/widgets/block_widget.tmpl"
		err          = widget.BlockDef.RenderForm(
			byteBuf, context.ID, context.Name,
			context.Value, errors, context,
		)
	)
	if err != nil {
		return errs.Wrap(err, "Error rendering block form")
	}

	context.BlockHTML = template.HTML(byteBuf.String())

	return tpl.FRender(w, context, templateName)
}

func RenderBlock(w io.Writer, block Block, value any, context ctx.Context) error {
	return block.Render(w, value, context)
}
