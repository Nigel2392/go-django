package pages

import (
	"net/http"
	"net/url"
	"slices"
	"strconv"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/permissions"
	"github.com/Nigel2392/mux"
)

type pageDefiner interface {
	attrs.Definer
	Page
}

func pageAdminAppHandler(fn func(http.ResponseWriter, *http.Request, *admin.AppDefinition, *admin.ModelDefinition)) mux.Handler {
	return mux.NewHandler(func(w http.ResponseWriter, req *http.Request) {
		var app, ok = admin.AdminSite.Apps.Get(AdminPagesAppName)
		if !ok {
			except.Fail(http.StatusBadRequest, "App not found")
			return
		}

		model, ok := app.Models.Get(AdminPagesModelPath)
		if !ok {
			except.Fail(http.StatusBadRequest, "Model %q not found in %v", AdminPagesModelPath, app.Models.Keys())
			return
		}

		fn(w, req, app, model)
	})
}

func pageHandler(fn func(http.ResponseWriter, *http.Request, *admin.AppDefinition, *admin.ModelDefinition, *PageNode)) mux.Handler {
	return mux.NewHandler(func(w http.ResponseWriter, req *http.Request) {

		var (
			routeVars = mux.Vars(req)
			pageID    = routeVars.GetInt(PageIDVariableName)
		)
		if pageID == 0 {
			except.Fail(http.StatusNotFound, "invalid page id")
			return
		}

		var qs = NewPageQuerySet().WithContext(req.Context())
		var page, err = qs.GetNodeByID(int64(pageID))
		if err != nil {
			except.Fail(http.StatusNotFound, "Failed to get page")
			return
		}

		var app django.AppConfig = pageApp
		if page.PageID != 0 && page.ContentType != "" {
			var pagesDef = DefinitionForType(page.ContentType)
			var appConf, ok = django.GetAppForModel(pagesDef.Object().(attrs.Definer))
			if !ok {
				logger.Error("Failed to get django.AppConfig for page type %q", page.ContentType)
				except.Fail(http.StatusNotFound, "App for page type not found")
				return
			}
			app = appConf
		}

		req = req.WithContext(django.ContextWithApp(
			req.Context(), app,
		))

		var handler = pageAdminAppHandler(func(w http.ResponseWriter, req *http.Request, app *admin.AppDefinition, model *admin.ModelDefinition) {
			fn(w, req, app, model, page)
		})

		handler.ServeHTTP(w, req)
	})
}

func getPageBreadcrumbs(r *http.Request, p *PageNode, urlForLast bool) ([]admin.BreadCrumb, error) {
	var breadcrumbs = make([]admin.BreadCrumb, 0, p.Depth+2)
	var qs = NewPageQuerySet().WithContext(r.Context())
	if p.Depth > 0 {
		var ancestors, err = qs.GetAncestors(
			p.Path, p.Depth,
		)
		if err != nil {
			return nil, err
		}
		slices.SortStableFunc(ancestors, func(a, b *PageNode) int {
			if a.Depth < b.Depth {
				return -1
			}
			if a.Depth > b.Depth {
				return 1
			}
			return 0
		})

		for _, a := range ancestors {
			a := a
			breadcrumbs = append(breadcrumbs, admin.BreadCrumb{
				Title: a.Title,
				URL:   django.Reverse("admin:pages:list", a.ID()),
			})
		}

		var b = admin.BreadCrumb{
			Title: p.Title,
		}

		if urlForLast {
			b.URL = django.Reverse("admin:pages:list", p.ID())
		}

		breadcrumbs = append(breadcrumbs, b)
	} else {
		breadcrumbs = append(breadcrumbs, admin.BreadCrumb{
			Title: trans.T(r.Context(), "Root Pages"),
			URL:   django.Reverse("admin:pages"),
		})
	}
	return breadcrumbs, nil
}

func getPageActions(rq *http.Request, p *PageNode) []admin.Action {
	var actions = make([]admin.Action, 0)
	if p.ID() == 0 {
		return actions
	}

	if p.StatusFlags.Is(StatusFlagPublished) {
		actions = append(actions, admin.Action{
			Icon:   "icon-view",
			Target: "_blank",
			Title:  trans.T(rq.Context(), "View Live"),
			URL:    URLPath(p),
		})
	}

	if django.AppInstalled("auditlogs") && permissions.HasPermission(rq, "auditlogs:list") {
		var u = django.Reverse(
			"admin:auditlogs",
		)

		var url, err = url.Parse(u)
		if err != nil {
			return actions
		}

		var q = url.Query()
		q.Set(
			"filters-object_id",
			strconv.Itoa(int(p.ID())),
		)
		q.Set(
			"filters-content_type",
			contenttypes.NewContentType(p).ShortTypeName(),
		)
		url.RawQuery = q.Encode()

		actions = append(actions, admin.Action{
			Icon:  "icon-history",
			Title: trans.T(rq.Context(), "History"),
			URL:   url.String(),
		})
	}

	return actions
}

func addNextUrl(current string, next string) string {
	if next == "" {
		return current
	}
	if current == "" {
		assert.Fail("current url is empty")
		return ""
	}

	var u, err = url.Parse(current)
	if err != nil {
		return current
	}
	var q = u.Query()
	q.Set("next", next)
	u.RawQuery = q.Encode()
	return u.String()
}
