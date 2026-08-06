package admin

import (
	"net/http"

	"github.com/Nigel2392/go-django/contrib/admin/searches"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/views"
)

var SearchView = &searches.SearchView{
	BaseView: views.BaseView{
		AllowedMethods:  []string{http.MethodGet},
		BaseTemplateKey: BASE_KEY,
		TemplateName:    "admin/views/search.tmpl",
	},
	FailURL: func(r *http.Request) string {
		return django.Reverse("admin:home")
	},
	GetSearchOption: func(r *http.Request, cTypeName string) *searches.SearchOptions {
		cType := contenttypes.DefinitionForType(cTypeName)
		model := FindDefinition(cType.ContentType().Model())

		if model == nil {
			return nil
		}

		return model.ListView.Search
	},
	GetSearchOptions: func(r *http.Request) []*searches.SearchOptions {
		return AdminSite.SearchableModels(r)
	},
	SetupOptions: func(r *http.Request, opts *searches.SearchOptions) {
		setupSearchWithModel(r, opts, FindDefinition(opts.ContentType()))
	},
	GetContext: func(r *http.Request, v *searches.BoundSearchView) ctx.ContextWithRequest {
		var context = NewContext(r, AdminSite, nil)
		var contextPage = PageOptions{
			TitleFn: trans.S(
				"Search %s", v.Search.PluralLabel(r.Context()),
			),
		}
		context.SetPage(contextPage)
		return context
	},
}
