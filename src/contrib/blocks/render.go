package blocks

import (
	"context"
	"io"

	"github.com/Nigel2392/go-django/src/core/ctx"
)

func RenderBlock(ctx context.Context, w io.Writer, block Block, value any, context ctx.Context) error {
	return block.Render(ctx, w, value, context)
}
