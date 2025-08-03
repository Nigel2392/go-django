package chooser

import (
	"context"
	"net/http"
	"reflect"

	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/views"
)

var (
	_ views.View         = (*ChooserFormView[attrs.Definer])(nil)
	_ views.MethodsView  = (*ChooserFormView[attrs.Definer])(nil)
	_ views.BindableView = (*ChooserFormView[attrs.Definer])(nil)
	_ views.Renderer     = (*BoundChooserFormView[attrs.Definer])(nil)
)

type ChooserFormView[T attrs.Definer] struct {
	Template       string
	AllowedMethods []string
	Panels         []admin.Panel
	Validate       []func(context.Context, *http.Request, T, *ChooserFormView[T]) error
	Save           func(context.Context, *http.Request, T, *ChooserFormView[T]) error

	_Definition *ChooserDefinition[T]
}

func (v *ChooserFormView[T]) ServeXXX(w http.ResponseWriter, req *http.Request) {
	// Placeholder method, will never get called.
}

func (v *ChooserFormView[T]) Methods() []string {
	return v.AllowedMethods
}

func (v *ChooserFormView[T]) Bind(w http.ResponseWriter, req *http.Request) (views.View, error) {
	var base = &BoundChooserFormView[T]{
		ChooserFormView: v,
		ResponseWriter:  w,
		Request:         req,
		Model: attrs.NewObject[T](
			reflect.TypeOf(v._Definition.Model),
		),
	}
	return base, nil
}
