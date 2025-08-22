package ghttp

import (
	"context"
	"net/http"

	"github.com/Nigel2392/go-django/src/components"
)

type ComponentHandler struct {
	Component components.Component
	Error     func(w http.ResponseWriter, r *http.Request, h *ComponentHandler, err error)
}

func Handler(c components.Component) *ComponentHandler {
	return &ComponentHandler{
		Component: c,
	}
}

type requestContextKey struct{}

func (h *ComponentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var ctx = context.WithValue(r.Context(), requestContextKey{}, h.Component)
	r = r.WithContext(ctx)
	if err := h.Component.Render(ctx, w); err != nil && h.Error != nil {
		h.Error(w, r, h, err)
	}
}
