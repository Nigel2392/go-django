package blocks

import (
	"io"

	"github.com/Nigel2392/go-django/src/core/ctx"
)

func RenderBlock(w io.Writer, block Block, value any, context ctx.Context) error {
	return block.Render(w, value, context)
}
