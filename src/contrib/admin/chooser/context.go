package chooser

import (
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/views"
)

type ModalContext struct {
	ctx.ContextWithRequest
	Definition Chooser
	Title      string
	View       views.View
}
