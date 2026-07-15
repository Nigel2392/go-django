package views

import (
	"net/http"

	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/mux"
)

var (
	_ View = (*BoundDetailView[any])(nil)

	_ ControlledView = (*DetailView[any])(nil)
	_ BindableView   = (*DetailView[any])(nil)
)

const (
	ErrMissingArg errs.Error = "missing url argument or argument is empty"
)

type detailview__ObjectGetter[T any] interface {
	GetObject(*http.Request, string) (T, error)
}

type detailview__ContextBinder[T any] interface {
	BindContext(ctx.Context, T)
}

type BoundDetailView[T any] struct {
	RW      http.ResponseWriter
	RQ      *http.Request
	View    DetailView[T]
	Object  T
	Context ctx.Context
}

func (b *BoundDetailView[T]) ServeXXX(http.ResponseWriter, *http.Request) {}

func (b *BoundDetailView[T]) BindContext(c ctx.Context, obj T) {
	b.Object = obj
	b.Context = c
}

type DetailView[T any] struct {
	BaseView
	ContextName     string
	URLArgName      string
	GetObjectFn     func(req *http.Request, urlArg string) (T, error)
	ChangeContextFn func(req *http.Request, object T, context ctx.ContextWithRequest) ctx.ContextWithRequest
	OnErrorFn       func(w http.ResponseWriter, r *http.Request, err error)
	PostMethod      func(d *DetailView[T], w http.ResponseWriter, r *http.Request, bound View) (http.ResponseWriter, *http.Request)
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

func (d *DetailView[T]) Bind(w http.ResponseWriter, req *http.Request) (View, error) {
	var bound = &BoundDetailView[T]{
		View: *d,
		RW:   w,
		RQ:   req,
	}
	return bound, nil
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

func (v *DetailView[T]) TakeControl(w http.ResponseWriter, r *http.Request, view View) {
	var err error
errCheck:
	if err != nil {
		v.OnError(w, r, err)
		return
	}

	urlArg, err := v.GetURLArg(r)
	if err != nil {
		goto errCheck
	}

	var object T
	if getter, ok := view.(detailview__ObjectGetter[T]); ok {
		object, err = getter.GetObject(r, urlArg)
	} else {
		object, err = v.GetObject(r, urlArg)
	}
	if err != nil {
		goto errCheck
	}

	var viewCtx ctx.Context
	if getter, ok := view.(ContextGetter); ok {
		viewCtx, err = getter.GetContext(r)
	} else {
		viewCtx, err = v.GetContext(r)
	}

	viewCtx.Set(v.ContextName, object)

	if binder, ok := view.(detailview__ContextBinder[T]); ok {
		binder.BindContext(viewCtx, object)
	}

	if v.ChangeContextFn != nil {
		viewCtx = v.ChangeContextFn(r, object, viewCtx.(ctx.ContextWithRequest))
	}

	if r.Method == http.MethodPost && v.PostMethod != nil {
		w, r = v.PostMethod(v, w, r, view)
		if w == nil || r == nil {
			return
		}
	}

	if err = TryServeTemplateView(w, r, []View{view, v}, viewCtx); err != nil {
		v.OnError(w, r, err)
	}
}

func (v *DetailView[T]) OnError(w http.ResponseWriter, r *http.Request, err error) {
	logger.Errorf(
		"Error while serving view: %v", err,
	)
	if v.OnErrorFn != nil {
		v.OnErrorFn(w, r, err)
	} else {
		except.Fail(
			http.StatusInternalServerError,
			"Error while serving view: %v", err,
		)
	}
}
