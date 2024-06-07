package views

import (
	"fmt"
	"net/http"
	"slices"

	"github.com/Nigel2392/django/core"
	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/core/ctx"
)

type ViewHandler = http.Handler

type View interface {
	// ServeXXX is a method that will never get called.
	// It is a placeholder for the actual method that will be called.
	// The actual method will be determined by the type of the request.
	// For example, if the request is a GET request, then ServeGET will be called.
	// If the request is a POST request, then ServePOST will be called.
	// etc...
	ServeXXX(w http.ResponseWriter, req *http.Request)
}

type MethodsView interface {
	View
	Methods() []string
}

type ContextGetter interface {
	GetContext(req *http.Request) (ctx.Context, error)
}

type TemplateView interface {
	View
	ContextGetter
	// GetTemplate returns the template that will be rendered.
	GetTemplate(req *http.Request) string
	// Render renders the template with the given context.
	Render(w http.ResponseWriter, req *http.Request, templateName string, context ctx.Context) error
}

type Renderer interface {
	View
	ContextGetter

	// Render renders the template with the given context.
	Render(w http.ResponseWriter, req *http.Request, context ctx.Context) error
}

type boundView struct {
	view           View
	allowedMethods []string
}

func (b *boundView) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var method = req.Method
	var serveFn, ok = attrs.Method[http.HandlerFunc](
		b.view, fmt.Sprintf("Serve%s", method),
	)

	if ok {
		serveFn(w, req)
		return
	}

	var methods []string
	if methodViews, ok := b.view.(MethodsView); ok {
		methods = methodViews.Methods()
	} else {
		methods = b.allowedMethods
	}

	if !slices.Contains(methods, method) {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var (
		context  ctx.Context
		template string
		err      error
	)

	if contextGetter, ok := b.view.(ContextGetter); ok {
		context, err = contextGetter.GetContext(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if templateView, ok := b.view.(TemplateView); ok {
		template = templateView.GetTemplate(req)
	}

	if context == nil {
		context = core.Context(req)
	}

	context.Set("Request", req)
	context.Set("Template", template)
	context.Set("View", b.view)

	if templateView, ok := b.view.(TemplateView); ok {
		if template == "" {
			goto renderer
		}

		err = templateView.Render(w, req, template, context)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}

renderer:
	if renderer, ok := b.view.(Renderer); ok {
		renderer.Render(w, req, context)
		return
	}
}

type BaseView struct {
}
