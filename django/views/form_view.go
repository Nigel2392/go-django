package views

import (
	"net/http"
	"reflect"

	"github.com/Nigel2392/django/core/ctx"
	"github.com/Nigel2392/django/forms"
)

var _ TemplateView = (*FormView[forms.Form])(nil)

type FormView[T forms.Form] struct {
	BaseView
	GetFormFn    func(req *http.Request) T
	GetInitialFn func(req *http.Request) map[string]interface{}
	ValidFn      func(req *http.Request, form T) error
	InvalidFn    func(req *http.Request, form T) error
}

type form[T forms.Form] struct {
	f T
}

func isZero(v interface{}) bool {
	if v == nil {
		return true
	}
	var rVal = reflect.ValueOf(v)
	return !rVal.IsValid() || rVal.Kind() == reflect.Ptr && rVal.IsNil()
}

func (v *FormView[T]) GetForm(req *http.Request) T {
	return v.GetFormFn(req)
}

func (v *FormView[T]) FormFromCtx(context ctx.Context) (t T) {
	var f = context.Get("form")
	if isZero(f) {
		return t
	}

	var form, ok = f.(*form[T])
	if !ok {
		return t
	}

	return form.f
}

func (v *FormView[T]) GetContext(req *http.Request) (ctx.Context, error) {
	var context, err = v.BaseView.GetContext(req)
	if err != nil {
		return nil, err
	}

	var f = v.GetForm(req)
	if isZero(f) {
		return context, nil
	}

	context.Set("form", &form[T]{f})

	return context, nil
}

func (v *FormView[T]) Render(w http.ResponseWriter, req *http.Request, templateName string, context ctx.Context) error {
	var form = v.FormFromCtx(context)
	if isZero(form) {
		return v.BaseView.Render(w, req, templateName, context)
	}

	form = forms.Initialize(
		form, forms.WithRequestData(http.MethodPost, req),
	)

	if v.GetInitialFn != nil {
		form.SetInitial(v.GetInitialFn(req))
	}

	if req.Method == http.MethodPost {

		if form.IsValid() {
			if v.ValidFn != nil {
				return v.ValidFn(req, form)
			}
		} else {
			if v.InvalidFn != nil {
				return v.InvalidFn(req, form)
			}
		}
	}

	context.Set("form", form)

	return v.BaseView.Render(w, req, templateName, context)
}
