package chooser

import (
	"context"

	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/views"
)

type ModalContext struct {
	ctx.ContextWithRequest
	Definition Chooser
	Title      any
	Errors     []error
	View       views.View
}

func (c *ModalContext) GetTitle() string {
	switch v := c.Title.(type) {
	case string:
		return v
	case func(context.Context) string:
		return v(c.Request().Context())
	}
	return ""
}
