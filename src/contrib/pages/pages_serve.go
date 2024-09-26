package pages

import (
	"context"
	"net/http"
	"strings"

	models "github.com/Nigel2392/go-django/src/contrib/pages/page_models"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/views"
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
			pageNotFound(w, req, nil, pathParts)
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
		pageNotFound(w, req, err, pathParts)
		return
	}

	if page.ID() == 0 {
		err = errors.New("No page found, ID is 0")
		pageNotFound(w, req, err, pathParts)
		return
	}

	if page.StatusFlags.Is(models.StatusFlagDeleted) ||
		page.StatusFlags.Is(models.StatusFlagHidden) ||
		!page.StatusFlags.Is(models.StatusFlagPublished) {
		err = errors.New("Page is not published")
		pageNotFound(w, req, err, pathParts)
		return
	}

	var definition = DefinitionForType(page.ContentType)
	if definition == nil {
		err = errors.New("No definition found for page")
		pageNotFound(w, req, err, pathParts)
		return
	}

	specific, err := Specific(
		context, page,
	)
	if err != nil {
		pageNotFound(w, req, err, pathParts)
		return
	}

	var view = definition.PageView()
	var handler, ok = specific.(http.Handler)

	if view == nil && !ok {
		err = errors.New("No view found for page")
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
