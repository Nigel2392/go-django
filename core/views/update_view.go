package views

import (
	"github.com/Nigel2392/go-django/core/views/interfaces"
	"github.com/Nigel2392/router/v3/request"
)

type UpdateView[T interfaces.Saver] struct {
	BaseFormView[T]
	Fields []string

	// GetQuerySet has priority over GetInstance!
	GetQuerySet func(r *request.Request) (T, error)
}

func (c *UpdateView[T]) ServeHTTP(r *request.Request) {
	c.BaseFormView.Action = "update"
	if len(c.Fields) > 0 {
		c.BaseFormView.WithFields(c.Fields...)
	}

	if c.GetQuerySet != nil {
		var instance, err = c.GetQuerySet(r)
		if err != nil {
			r.Error(500, err.Error())
			return
		}
		c.BaseFormView.GetInstance = func(r *request.Request) T {
			return instance
		}
	}
	c.BaseFormView.Serve(r)
}
