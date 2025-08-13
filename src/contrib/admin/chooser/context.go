package chooser

import (
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/trans"
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
	var t, _ = trans.GetText(c.Request().Context(), c.Title)
	return t
}
