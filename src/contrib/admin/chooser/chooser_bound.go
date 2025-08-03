package chooser

import (
	"net/http"

	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
)

type BoundChooserFormPage[T attrs.Definer] struct {
	FormView       *ChooserFormPage[T]
	ResponseWriter http.ResponseWriter
	Request        *http.Request
	Model          T
}

func (v *BoundChooserFormPage[T]) ServeXXX(w http.ResponseWriter, req *http.Request) {
	// Placeholder method, will never get called.
}

func (v *BoundChooserFormPage[T]) GetContext(req *http.Request) (ctx.Context, error) {
	var c = ctx.RequestContext(req)

	return c, nil
}

func (v *BoundChooserFormPage[T]) Render(w http.ResponseWriter, req *http.Request, context ctx.Context) error {
	return nil
}
