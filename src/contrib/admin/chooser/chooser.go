package chooser

import (
	"context"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

type ChooserDefinition[T attrs.Definer] struct {
	Model    T
	Title    any // string or func(ctx context.Context) string
	Template string

	ListView   *ChooserListView[T]
	CreateView *ChooserFormView[T]
	UpdateView *ChooserFormView[T]
}

func (c *ChooserDefinition[T]) GetModel() attrs.Definer {
	return c.Model
}

func (c *ChooserDefinition[T]) GetTitle(ctx context.Context) string {
	switch v := c.Title.(type) {
	case string:
		return v
	case func(ctx context.Context) string:
		return v(ctx)
	}
	assert.Fail("ChooserDefinition.Title must be a string or a function that returns a string")
	return ""
}
