package views

import (
	"github.com/Nigel2392/go-django/core/views/interfaces"
	"github.com/Nigel2392/router/v3/request"
)

type CreateView[T interfaces.Saver] struct {
	BaseFormView[T]
	Fields []string
}

func (c *CreateView[T]) ServeHTTP(r *request.Request) {
	c.BaseFormView.Action = "create"
	if len(c.Fields) > 0 {
		c.BaseFormView.WithFields(c.Fields...)
	}
	c.BaseFormView.Serve(r)
}
