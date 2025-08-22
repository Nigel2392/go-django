package gluecomps

import (
	"context"
	"io"

	"github.com/Nigel2392/go-django/src/components"
)

// Glue is a package for gluing together standard go http and go-django functionality with components

type fn struct {
	fn func(ctx context.Context, w io.Writer) error
}

func Func(f func(ctx context.Context, w io.Writer) error) components.Component {
	return &fn{fn: f}
}

func (f *fn) Render(ctx context.Context, w io.Writer) error {
	return f.fn(ctx, w)
}
