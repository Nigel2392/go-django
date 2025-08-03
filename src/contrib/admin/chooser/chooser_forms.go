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
	_ views.View         = (*ChooserFormPage[attrs.Definer])(nil)
	_ views.MethodsView  = (*ChooserFormPage[attrs.Definer])(nil)
	_ views.BindableView = (*ChooserFormPage[attrs.Definer])(nil)
	_ views.Renderer     = (*BoundChooserFormPage[attrs.Definer])(nil)
)

type ChooserFormPage[T attrs.Definer] struct {
	Template       string
	AllowedMethods []string
	Panels         []admin.Panel
	Validate       []func(context.Context, *http.Request, T, *ChooserFormPage[T]) error
	Save           func(context.Context, *http.Request, T, *ChooserFormPage[T]) error

	_Definition *ChooserDefinition[T]
}

func (v *ChooserFormPage[T]) ServeXXX(w http.ResponseWriter, req *http.Request) {
	// Placeholder method, will never get called.
}

func (v *ChooserFormPage[T]) Methods() []string {
	return v.AllowedMethods
}

func (v *ChooserFormPage[T]) Bind(w http.ResponseWriter, req *http.Request) (views.View, error) {
	var base = &BoundChooserFormPage[T]{
		FormView:       v,
		ResponseWriter: w,
		Request:        req,
		Model: attrs.NewObject[T](
			reflect.TypeOf(v._Definition.Model),
		),
	}
	return base, nil
}
