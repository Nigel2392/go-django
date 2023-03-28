package app

import (
	"github.com/Nigel2392/router/v3"
	"github.com/Nigel2392/router/v3/request"
)

func LoggerMiddleware(l request.Logger) router.Middleware {
	return func(next router.Handler) router.Handler {
		return router.HandleFunc(func(r *request.Request) {
			r.Logger = l
			next.ServeHTTP(r)
		})
	}
}
