package views

import (
	"fmt"
	"net/http"
	"slices"

	"github.com/Nigel2392/django"
	"github.com/Nigel2392/django/core"
	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/django/core/attrs"
	"github.com/Nigel2392/django/core/ctx"
	"github.com/Nigel2392/django/core/errs"
	"github.com/Nigel2392/django/core/tpl"
)

var httpMethods = []string{
	http.MethodGet,
	http.MethodHead,
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
	http.MethodDelete,
	http.MethodConnect,
	http.MethodOptions,
	http.MethodTrace,
}

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
	View
	GetContext(req *http.Request) (ctx.Context, error)
}

type TemplateGetter interface {
	// GetTemplate returns the template that will be rendered.
	GetTemplate(req *http.Request) string
}

type TemplateView interface {
	View
	ContextGetter
	TemplateGetter
	// Render renders the template with the given context.
	Render(w http.ResponseWriter, req *http.Request, templateName string, context ctx.Context) error
}

type ErrorHandler interface {
	// HandleError handles the error.
	HandleError(w http.ResponseWriter, req *http.Request, err error, code int)
}

type Renderer interface {
	View
	ContextGetter

	// Render renders the template with the given context.
	Render(w http.ResponseWriter, req *http.Request, context ctx.Context) error
}

type TemplateKeyer interface {
	// GetBaseKey returns the base key for the template.
	GetBaseKey() string
}

type ErrorFunc func(w http.ResponseWriter, req *http.Request, err error, code int)

// Serve serves a view.
//
// This function is a generic view handler that can be used
// to serve any type of view which the `Invoke` function can handle.
func Serve(view View) http.Handler {
	var allowedFnMethods []string
	for _, method := range httpMethods {
		if _, ok := attrs.Method[http.HandlerFunc](
			view, fmt.Sprintf("Serve%s", method),
		); ok {
			allowedFnMethods = append(allowedFnMethods, method)
		}
	}

	if methodNamer, ok := view.(MethodsView); ok {
		for _, method := range methodNamer.Methods() {
			if !slices.Contains(allowedFnMethods, method) {
				allowedFnMethods = append(allowedFnMethods, method)
			}
		}
	}

	assert.True(
		len(allowedFnMethods) > 0,
		"View must have at least one Serve method defined, I.E. ServeGET, ServePOST, etc...",
	)

	var allowedMethods []string
	if methodViews, ok := view.(MethodsView); ok {
		var m = methodViews.Methods()
		allowedMethods = make([]string, 0, len(m)+len(allowedMethods))
		for _, method := range m {
			if !slices.Contains(allowedFnMethods, method) {
				allowedMethods = append(allowedMethods, method)
			}
		}
	} else {
		allowedMethods = allowedFnMethods
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		Invoke(view, w, req, allowedMethods...)
	})
}

func handleErrors(w http.ResponseWriter, req *http.Request, err error, code int) {
	http.Error(w, "Error processing request", code)
}

// Invoke invokes the view and appropriately handles the request.
//
// This view is a generic view that can be used to render templates.
//
// It is moved into a separate function to better handle views and their composition.
//
// Example:
//
//	type MyView struct {
//		TemplateView // Some sort of base view with a get context method.
//		// Fields here...
//	}
//
//	func (v *MyView) GetContext(req *http.Request) (ctx.Context, error) {
//		// Get the context here...
//	}
//
// In regular GO, the above TemplateView would not be able to get the context of the 'MyView' struct.
//
// This means the overridden GetContext method would remain completely invisible and thus go unused.
//
// This prevents a whole lot of flexibility in the design of the views.
//
// This function is a workaround to allow for a more class-based approach.
//
// It can handle the following types of views:
//
//  1. View - A generic view that can handle any type of request.
//  2. MethodsView - A view that declares acceptable methods.
//  3. ContextGetter - A view that can initiate a context for the request.
//  4. TemplateGetter - A view that can get the template path of the template to render.
//  5. TemplateView - A view that can render a template from a template path with a context.
//  6. ErrorHandler - A view that can handle errors.
//  7. Renderer - A view that can render directly to the response writer with a context.
//  8. TemplateKeyer - A view that can get the base key for the template.
//     This is useful for rendering a sub-template with a base template.
func Invoke(view View, w http.ResponseWriter, req *http.Request, allowedMethods ...string) {
	var method = req.Method

	// Setup error handling.
	var errFn = handleErrors
	if errHandler, ok := view.(ErrorHandler); ok {
		errFn = errHandler.HandleError
	}

	// Check if the method is allowed.
	if len(allowedMethods) > 0 && !slices.Contains(allowedMethods, method) {
		django.App().Log.Error("Method not allowed")
		errFn(
			w, req,
			errs.Error("Method not allowed"),
			http.StatusMethodNotAllowed,
		)
		return
	}

	// Check if the view has a Serve<XXX> method.
	var serveFn, ok = attrs.Method[http.HandlerFunc](
		view, fmt.Sprintf("Serve%s", method),
	)
	if ok {
		// Any matching serve method takes precedence over the fallback.
		serveFn(w, req)
		return
	}

	var (
		context  ctx.Context
		baseKey  string
		template string
		err      error
	)

	// Get the context if the view implements the ContextGetter interface.
	if contextGetter, ok := view.(ContextGetter); ok {
		if context, err = contextGetter.GetContext(req); err != nil {
			django.App().Log.Error(err)
			errFn(w, req, err, http.StatusInternalServerError)
			return
		}
	}

	// If the context is nil, then create a new context.
	if context == nil {
		context = core.Context(req)
	}

	// Set basic context variables.
	context.Set("Request", req)
	context.Set("Template", template)
	context.Set("View", view)

	// Render the template immediately if the view implements the Renderer interface.
	if renderer, ok := view.(Renderer); ok {
		if err = renderer.Render(w, req, context); err != nil {
			django.App().Log.Error(err)
			errFn(w, req, err, http.StatusInternalServerError)
		}
		return
	}

	// Get the template if the view implements the TemplateView interface.
	if templateView, ok := view.(TemplateView); ok {
		template = templateView.GetTemplate(req)
	}

	// Get the base key if the view implements the TemplateKeyer interface.
	// This is to render the proper base template for the sub-template (template inheritance.)
	if templateKeyer, ok := view.(TemplateKeyer); ok {
		baseKey = templateKeyer.GetBaseKey()
	}

	// Render the template if the view implements the TemplateView interface.
	if templateView, ok := view.(TemplateView); ok {
		if err = templateView.Render(w, req, template, context); err != nil {
			django.App().Log.Error(err)
			errFn(w, req, err, http.StatusInternalServerError)
		}
		return
	}

	// Cannot render if there is no template.
	// Developer error - HARD FAIL.
	assert.False(
		template == "" && baseKey == "",
		"Template and base key cannot be empty",
	)

	// Render the template.
	// This has to be a switch statement
	// because of the way the tpl package is designed.
	switch {
	case template != "" && baseKey != "":
		err = tpl.FRender(w, context, baseKey, template)
	case template != "":
		err = tpl.FRender(w, context, template)
	case baseKey != "":
		err = tpl.FRender(w, context, baseKey)
	}
	if err != nil {
		django.App().Log.Error(err)
		errFn(w, req, err, http.StatusInternalServerError)
	}
}
