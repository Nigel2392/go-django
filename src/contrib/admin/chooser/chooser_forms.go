package chooser

import (
	"context"
	"net/http"
	"reflect"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
	"github.com/Nigel2392/go-django/src/models"
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
	Validate       []func(context.Context, *http.Request, T, *BoundChooserFormPage[T]) error
	Save           func(context.Context, *http.Request, T, *BoundChooserFormPage[T]) error

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
		View:           v,
		ResponseWriter: w,
		Request:        req,
		Model: attrs.NewObject[T](
			reflect.TypeOf(v._Definition.Model),
		),
	}
	return base, nil
}

func (v *ChooserFormPage[T]) GetContext(req *http.Request, bound *BoundChooserFormPage[T]) ctx.Context {
	var c = v._Definition.GetContext(req, v, bound)

	return c
}

type BoundChooserFormPage[T attrs.Definer] struct {
	View           *ChooserFormPage[T]
	ResponseWriter http.ResponseWriter
	Request        *http.Request
	Model          T
}

func (v *BoundChooserFormPage[T]) ServeXXX(w http.ResponseWriter, req *http.Request) {
	// Placeholder method, will never get called.
}

func (v *BoundChooserFormPage[T]) GetPanels() []admin.Panel {
	if v.View.Panels != nil {
		return v.View.Panels
	}
	return []admin.Panel{}
}

func (v *BoundChooserFormPage[T]) GetForm(req *http.Request) *admin.AdminModelForm[modelforms.ModelForm[T], T] {

	var form = modelforms.NewBaseModelForm(req.Context(), v.Model)
	form.SaveInstance = func(ctx context.Context, t T) error {
		if v.View.Save != nil {
			return v.View.Save(ctx, req, t, v)
		}

		saved, err := models.SaveModel(ctx, t)
		if err != nil {
			return err
		}
		if !saved {
			return errors.NoChanges.Wrap("model not saved, no changes made")
		}

		return nil
	}

	var adminForm = admin.NewAdminModelForm[modelforms.ModelForm[T]](
		form, v.GetPanels()...,
	)

	adminForm.Load()

	return adminForm
}

func (v *BoundChooserFormPage[T]) GetContext(req *http.Request) (ctx.Context, error) {
	var c = v.View.GetContext(req, v)
	c.Set("form", v.GetForm(req))
	return c, nil
}

func (v *BoundChooserFormPage[T]) Render(w http.ResponseWriter, req *http.Request, context ctx.Context) error {
	var form = context.Get("form").(*admin.AdminModelForm[modelforms.ModelForm[T], T])
	_ = form
	return nil
}
