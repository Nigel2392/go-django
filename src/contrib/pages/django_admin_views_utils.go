package pages

import (
	"net/http"
	"net/url"
	"path"
	"slices"
	"strconv"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	models "github.com/Nigel2392/go-django/src/contrib/pages/page_models"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/forms/fields"
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
			except.Fail(http.StatusBadRequest, "Model not found")
			return
		}

		fn(w, req, app, model)
	})
}

func pageHandler(fn func(http.ResponseWriter, *http.Request, *admin.AppDefinition, *admin.ModelDefinition, *models.PageNode)) mux.Handler {
	return mux.NewHandler(func(w http.ResponseWriter, req *http.Request) {

		var (
			ctx       = req.Context()
			routeVars = mux.Vars(req)
			pageID    = routeVars.GetInt("page_id")
		)
		if pageID == 0 {
			except.Fail(http.StatusNotFound, "invalid page id")
			return
		}

		var page, err = QuerySet().GetNodeByID(ctx, int64(pageID))
		if err != nil {
			except.Fail(http.StatusNotFound, "Failed to get page")
			return
		}

		var handler = pageAdminAppHandler(func(w http.ResponseWriter, req *http.Request, app *admin.AppDefinition, model *admin.ModelDefinition) {
			fn(w, req, app, model, &page)
		})

		handler.ServeHTTP(w, req)
	})
}

func getPageBreadcrumbs(r *http.Request, p *models.PageNode, urlForLast bool) ([]admin.BreadCrumb, error) {
	var breadcrumbs = make([]admin.BreadCrumb, 0, p.Depth+2)
	if p.Depth > 0 {
		var ancestors, err = AncestorNodes(
			QuerySet(), r.Context(), p.Path, int(p.Depth)+1,
		)
		if err != nil {
			return nil, err
		}
		slices.SortStableFunc(ancestors, func(a, b models.PageNode) int {
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
			Title: "Root Pages",
			URL:   django.Reverse("admin:pages"),
		})
	}
	return breadcrumbs, nil
}

func getPageActions(rq *http.Request, p *models.PageNode) []admin.Action {
	var actions = make([]admin.Action, 0)
	if p.ID() == 0 {
		return actions
	}

	if p.StatusFlags.Is(models.StatusFlagPublished) {
		actions = append(actions, admin.Action{
			Icon:   "icon-view",
			Target: "_blank",
			Title:  fields.T("View Live"),
			URL: path.Join(
				pageApp.routePrefix, p.UrlPath,
			),
		})
	}

	if django.AppInstalled("auditlogs") &&
		permissions.HasPermission(rq, "auditlogs:list") {
		var u = django.Reverse(
			"admin:auditlogs",
		)

		var url, err = url.Parse(u)
		if err != nil {
			return actions
		}

		var q = url.Query()
		q.Set(
			"object_id",
			strconv.Itoa(int(p.ID())),
		)
		q.Set(
			"content_type",
			contenttypes.NewContentType(p).ShortTypeName(),
		)
		url.RawQuery = q.Encode()

		actions = append(actions, admin.Action{
			Icon:  "icon-history",
			Title: fields.T("History"),
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
