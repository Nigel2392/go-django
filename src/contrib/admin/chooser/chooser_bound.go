package chooser

import (
	"net/http"

	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
)

type BoundChooserFormView[T attrs.Definer] struct {
	ChooserFormView *ChooserFormView[T]
	ResponseWriter  http.ResponseWriter
	Request         *http.Request
	Model           T
}

func (v *BoundChooserFormView[T]) ServeXXX(w http.ResponseWriter, req *http.Request) {
	// Placeholder method, will never get called.
}

func (v *BoundChooserFormView[T]) GetContext(req *http.Request) (ctx.Context, error) {
	var c = ctx.RequestContext(req)

	return c, nil
}

func (v *BoundChooserFormView[T]) Render(w http.ResponseWriter, req *http.Request, context ctx.Context) error {
	return nil
}
