package pages

import (
	"net/http"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/mux"
	"github.com/a-h/templ"
)

var (
	_ views.ControlledView = (*PageServeView)(nil)
)

//
//	func newPathParts(path string) []string {
//		path = strings.Trim(path, "/")
//		var split = strings.Split(path, "/")
//		var parts []string = make([]string, 0, len(split))
//		for _, part := range split {
//			if part != "" {
//				parts = append(parts, part)
//			}
//		}
//		return parts
//	}

func Serve(allowedMethods ...string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var view = &PageServeView{}
		if err := views.Invoke(view, w, r, allowedMethods...); err != nil {
			logger.Errorf("Error invoking view: %v", err)
		}
	})
}

// A wrapper interface to get the same dynamic functionality as
// the views.View interface.
//
// This handles "PageView" objects much like the views.Invoke method.
type PageViewHandler interface {
	views.MethodsView
	views.ControlledView
	View() PageView
	CurrentPage() Page
}

type PageView interface {
	// ServePage is a method that will never get called.
	// It acts like the views.ServeXXX method.
	ServePage()
}

type PageContextGetter interface {
	GetContext(req *http.Request, page Page) (ctx.Context, error)
}

type PageBaseKeyGetter interface {
	GetBaseKey(req *http.Request, page Page) string
}

type PageTemplateGetter interface {
	GetTemplate(req *http.Request, page Page) string
}

type PageTemplateView interface {
	PageContextGetter
	PageTemplateGetter
}

type PageRenderView interface {
	Render(w http.ResponseWriter, req *http.Request, page Page, context ctx.Context) error
}

type PageTemplateRenderView interface {
	PageTemplateView
	Render(w http.ResponseWriter, req *http.Request, template string, page Page, context ctx.Context) error
}

type PageComponentView interface {
	PageContextGetter
	Render(req *http.Request, page Page) templ.Component
}

// Register to the page definition.
type PageServeView struct{}

func (v *PageServeView) ServeXXX(w http.ResponseWriter, req *http.Request) {}

func (v *PageServeView) TakeControl(w http.ResponseWriter, r *http.Request) {
	var pathParts = mux.Vars(r).GetAll("*")
	var req, site, err = SiteForRequest(r)
	if err != nil {
		pageNotFound(w, req, err, pathParts)
		return
	}

	if len(pathParts) > 0 && pathParts[0] == site.Root.Slug {
		pathParts = pathParts[1:]
	}

	pageObject, err := site.Root.Route(req, pathParts)
	if err != nil {
		pageNotFound(w, req, err, pathParts)
		return
	}

	if pageObject.ID() == 0 {
		pageNotFound(w, req, ErrNoPageID, pathParts)
		return
	}

	var page = pageObject.Reference()
	if page.StatusFlags.Is(StatusFlagDeleted) ||
		page.StatusFlags.Is(StatusFlagHidden) ||
		!page.StatusFlags.Is(StatusFlagPublished) {
		err = errors.CheckFailed.Wrap("Page is not published")
		pageNotFound(w, req, err, pathParts)
		return
	}

	specific, err := page.Specific(req.Context())
	if err != nil {
		pageNotFound(w, req, err, pathParts)
		return
	}

	var definition = DefinitionForType(page.ContentType)
	if definition == nil {
		err = errors.NotImplemented.Wrap("No definition found for page")
		pageNotFound(w, req, err, pathParts)
		return
	}

	var view = definition.PageView(specific)
	var handler, ok = specific.(http.Handler)
	if view == nil && !ok {
		err = errors.NotImplemented.Wrap("No view found for page")
		pageNotFound(w, req, err, pathParts)
		return
	}

	if ok {
		handler.ServeHTTP(w, req)
		return
	}

	var viewCtx ctx.Context
	if c, ok := view.(PageContextGetter); ok {
		viewCtx, err = c.GetContext(req, specific)
	}
	if err != nil {
		// http.Error(w, "Internal server error", http.StatusInternalServerError)
		logger.Errorf("Error getting context: %v", err)
		return
	}

	if viewCtx == nil {
		viewCtx = ctx.RequestContext(req)
	}

	viewCtx.Set("Object", specific)
	viewCtx.Set("View", view)
	viewCtx.Set("Node", page)

	var template string
	if c, ok := view.(PageTemplateGetter); ok {
		template = c.GetTemplate(req, specific)
	}

	var baseKey string
	if c, ok := view.(PageBaseKeyGetter); ok {
		baseKey = c.GetBaseKey(req, specific)
	}

	var rView, rViewOk = view.(PageRenderView)
	if r, ok := view.(PageTemplateRenderView); ok {

		if baseKey == "" && template == "" && !rViewOk {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			logger.Errorf("No template or base key provided for rendering, nor a plain render method found")
			return
		}

		if baseKey == "" && template == "" {
			goto renderView
		}

		if err := r.Render(w, req, template, specific, viewCtx); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			logger.Errorf("Error rendering page: %v", err)
			return
		}
		return
	}

renderView:
	if rViewOk {
		if err := rView.Render(w, req, specific, viewCtx); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			logger.Errorf("Error rendering page: %v", err)
			return
		}
		return
	}

	http.Error(w, "Internal server error", http.StatusInternalServerError)
	logger.Errorf("No render method found for page")
}

func pageNotFound(_ http.ResponseWriter, _ *http.Request, err error, pathParts []string) {
	if err != nil {
		logger.Errorf("Error retrieving page: %v (%v)", err, pathParts)
	}
	except.Fail(
		http.StatusNotFound,
		"Page not found: %v", pathParts,
	)
}
