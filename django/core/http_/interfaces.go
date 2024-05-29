package http_

import "github.com/Nigel2392/mux"

type Mux interface {
	Use(middleware ...mux.Middleware)
	Handle(method string, path string, handler mux.Handler, name ...string) *mux.Route
	AddRoute(route *mux.Route)
}

type URL interface {
	Register(mux Mux)
}

type Middleware interface {
	URL
}

type MiddlewareImpl struct {
	Handler mux.Middleware
}

func (m *MiddlewareImpl) Register(mux Mux) {
	mux.Use(m.Handler)
}

func NewMiddleware(handler mux.Middleware) Middleware {
	return &MiddlewareImpl{Handler: handler}
}
