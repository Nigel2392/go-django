package views

import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/Nigel2392/django/core/ctx"
	"github.com/Nigel2392/django/forms"
)

var _ TemplateView = (*FormView[forms.Form])(nil)
var _ Checker = (*FormView[forms.Form])(nil)

type FormView[T forms.Form] struct {
	BaseView
	GetFormFn        func(req *http.Request) T
	GetInitialFn     func(req *http.Request) map[string]interface{}
	ValidFn          func(req *http.Request, form T) error
	InvalidFn        func(req *http.Request, form T) error
	SuccessFn        func(w http.ResponseWriter, req *http.Request, form T)
	CheckPermissions func(w http.ResponseWriter, req *http.Request) error
	FailsPermissions func(w http.ResponseWriter, req *http.Request, err error)
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

func (v *FormView[T]) Check(w http.ResponseWriter, req *http.Request) error {
	if v.CheckPermissions != nil {
		return v.CheckPermissions(w, req)
	}
	return nil
}

func (v *FormView[T]) Fail(w http.ResponseWriter, req *http.Request, err error) {
	if v.FailsPermissions != nil {
		v.FailsPermissions(w, req, err)
		return
	}
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
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

	var err error
	if req.Method == http.MethodPost {
		if form.IsValid() {

			if saver, ok := any(form).(interface{ Save() error }); ok {
				err = saver.Save()
				if err != nil {
					if v.InvalidFn != nil {
						err = v.InvalidFn(req, form)
						goto checkFormErr
					} else {
						if adder, ok := any(form).(forms.ErrorAdder); ok {
							adder.AddFormError(err)
						}
					}
					goto render
				}
			}

			if v.ValidFn != nil {
				err = v.ValidFn(req, form)
				if err != nil {
					goto checkFormErr
				}
			}

			if v.SuccessFn != nil {
				v.SuccessFn(w, req, form)
				return nil
			}

		} else {
			if v.InvalidFn != nil {
				err = v.InvalidFn(req, form)
			}
		}
	}

checkFormErr:
	if err != nil {
		if adder, ok := any(form).(forms.ErrorAdder); ok {
			adder.AddFormError(err)
		} else {
			return fmt.Errorf("form error: %w", err)
		}
	}

render:
	context.Set("form", form)

	return v.BaseView.Render(w, req, templateName, context)
}
