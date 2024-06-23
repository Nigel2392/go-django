package pages

import (
	"context"
	"net/http"
	"strings"

	"github.com/Nigel2392/django/contrib/pages/models"
	"github.com/Nigel2392/django/core"
	"github.com/Nigel2392/django/core/ctx"
	"github.com/Nigel2392/django/core/logger"
	"github.com/Nigel2392/django/views"
	"github.com/pkg/errors"
)

var (
	_ views.ControlledView = (*PageServeView)(nil)
)

func newPathParts(path string) []string {
	path = strings.Trim(path, "/")
	return strings.Split(path, "/")
}

func Serve(allowedMethods ...string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var view = &PageServeView{}
		if err := views.Invoke(view, w, r, allowedMethods...); err != nil {
			logger.Errorf("Error invoking view: %v\n", err)
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

// Register to the page definition.
type PageServeView struct{}

func (v *PageServeView) ServeXXX(w http.ResponseWriter, req *http.Request) {}

func (v *PageServeView) TakeControl(w http.ResponseWriter, req *http.Request) {

	var (
		context   = context.Background()
		pathParts = newPathParts(req.URL.Path)
		querySet  = QuerySet()
		page      models.PageNode
		err       error
	)

	if len(pathParts) == 0 {
		var pages, err = querySet.GetNodesByDepth(context, 0, 1000, 0)
		if err != nil {
			goto checkError
		}
		if len(pages) == 0 {
			http.Error(w, "No pages have been created", http.StatusNotFound)
			return
		}

		page = pages[0]
	} else {
		var p models.PageNode
		for i, part := range pathParts {
			p, err = querySet.GetNodeBySlug(context, part, int64(i), page.Path)
			if err != nil {
				err = errors.Wrapf(
					err, "Error getting page by slug (%d): %s/%s", i, page.Path, part,
				)
				break
			}
			page = p
		}
	}

checkError:
	if err != nil {
		http.Error(w, "Page not found", http.StatusNotFound)
		logger.Errorf("Error retrieving page: %v (%v)\n", err, pathParts)
		return
	}

	if page.ID() == 0 {
		http.Error(w, "Page not found", http.StatusNotFound)
		return
	}

	var definition = DefinitionForType(page.ContentType)
	if definition == nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		logger.Errorf("No definition found for page: %+v\n", page)
		return
	}

	specific, err := Specific(
		context, page,
	)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		logger.Errorf("Error getting specific page: %v\n", err)
		return
	}

	var view = definition.PageView()

	if view == nil {
		logger.Fatalf(
			500, "view is nil, cannot serve page",
		)
	}

	var viewCtx ctx.Context
	if c, ok := view.(PageContextGetter); ok {
		viewCtx, err = c.GetContext(req, specific)
	}
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		logger.Errorf("Error getting context: %v\n", err)
		return
	}

	if viewCtx == nil {
		viewCtx = core.Context(req)
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
			logger.Errorf("No template or base key provided for rendering, nor a plain render method found\n")
			return
		}

		if baseKey == "" && template == "" {
			goto renderView
		}

		if err := r.Render(w, req, template, specific, viewCtx); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			logger.Errorf("Error rendering page: %v\n", err)
			return
		}
		return
	}

renderView:
	if rViewOk {
		if err := rView.Render(w, req, specific, viewCtx); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			logger.Errorf("Error rendering page: %v\n", err)
			return
		}
		return
	}

	http.Error(w, "Internal server error", http.StatusInternalServerError)
	logger.Errorf("No render method found for view (%T)\n", view)
}
