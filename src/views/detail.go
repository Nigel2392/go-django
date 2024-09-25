package views

import (
	"net/http"

	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/mux"
)

const (
	ErrMissingArg errs.Error = "missing url argument or argument is empty"
)

type DetailView[T any] struct {
	BaseView
	ContextName string
	URLArgName  string
	GetObjectFn func(req *http.Request, urlArg string) (T, error)
}

func (d *DetailView[T]) Setup(w http.ResponseWriter, req *http.Request) (http.ResponseWriter, *http.Request) {
	if d.ContextName == "" {
		d.ContextName = "object"
	}
	if d.URLArgName == "" {
		d.URLArgName = "primary"
	}
	return w, req
}

func (d *DetailView[T]) GetObject(req *http.Request, urlArg string) (v T, err error) {
	if d.GetObjectFn != nil {
		return d.GetObjectFn(req, urlArg)
	}
	return v, nil
}

func (d *DetailView[T]) GetURLArg(req *http.Request) (string, error) {
	var vars = mux.Vars(req)
	var urlArg = vars.Get(d.URLArgName)
	if urlArg == "" {
		return "", ErrMissingArg
	}
	return urlArg, nil
}

func (d *DetailView[T]) GetContext(req *http.Request) (ctx.Context, error) {
	context, err := d.BaseView.GetContext(req)
	if err != nil {
		return nil, err
	}

	urlArg, err := d.GetURLArg(req)
	if err != nil {
		return nil, err
	}

	obj, err := d.GetObject(req, urlArg)
	if err != nil {
		return nil, err
	}

	context.Set(d.ContextName, obj)
	return context, nil
}
