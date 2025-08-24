package admin

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/messages"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/pagination"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/go-django/src/views/list"
)

var (
	_ views.View         = &SearchView{}
	_ views.BindableView = &SearchView{}
)

type SearchView struct{}

func (s *SearchView) ServeXXX(w http.ResponseWriter, req *http.Request) {
}

func (s *SearchView) Methods() []string {
	return []string{http.MethodGet}
}

func (s *SearchView) Bind(w http.ResponseWriter, req *http.Request) (views.View, error) {
	var v = &BoundSearchView{
		BaseView: views.BaseView{
			AllowedMethods:  []string{http.MethodGet},
			BaseTemplateKey: BASE_KEY,
			TemplateName:    "admin/views/search.tmpl",
		},
	}
	return v, nil
}

type BoundSearchView struct {
	views.BaseView
	W     http.ResponseWriter
	R     *http.Request
	Model *ModelDefinition
}

func (b *BoundSearchView) Setup(w http.ResponseWriter, req *http.Request) (http.ResponseWriter, *http.Request) {
	b.W = w
	b.R = req

	var cTypeName = req.URL.Query().Get("content_type")
	if cTypeName != "" {
		var cType = contenttypes.DefinitionForType(cTypeName)
		b.Model = FindDefinition(cType.ContentType().Model())
		if b.Model == nil && cType != nil {
			except.Fail(
				http.StatusNotFound,
				"model %T not found for adminsite",
				cTypeName,
			)
			return nil, nil
		}
	} else {
		var defs = AdminSite.SearchableModels(req)
		if len(defs) == 0 {
			except.Fail(
				http.StatusNotFound,
				"no searchable models found for adminsite",
				nil,
			)
			return nil, nil
		}

		b.Model = defs[0]
	}

	var searchOpts = b.Model.ListView.Search
	if searchOpts == nil || !searchOpts.CanSearch(req) {
		messages.Error(
			req, trans.T(req.Context(), "Search is not allowed for this model"),
		)
		http.Redirect(
			w,
			req,
			django.Reverse("admin:home"),
			http.StatusFound,
		)
		return nil, nil
	}

	return w, req
}

func (b *BoundSearchView) GetList(v *BoundSearchView, objects []attrs.Definer, columns []list.ListColumn[attrs.Definer]) (StringRenderer, error) {
	if b.Model.ListView.Search.GetList != nil {
		return b.Model.ListView.Search.GetList(b, objects, columns)
	}
	return list.NewList(v.R, b.Model.NewInstance(), objects, columns...), nil
}

func (b *BoundSearchView) GetContext(req *http.Request) (ctx.Context, error) {
	var context = NewContext(req, AdminSite, nil)
	var searchQuery = req.URL.Query().Get("global-search")

	var fields = b.Model.ListView.Search.ListFields
	if len(fields) == 0 {
		fields = b.Model.ListView.Fields
	}

	var columns = make([]list.ListColumn[attrs.Definer], len(fields))
	for i, field := range fields {
		columns[i] = b.Model.GetColumn(
			req.Context(), b.Model.ListView, field,
		)
	}

	var amount = b.Model.ListView.Search.PerPage
	if amount == 0 {
		amount = int(b.Model.ListView.PerPage)
	}
	if amount == 0 {
		amount = 25
	}

	var qs *queries.QuerySet[attrs.Definer]
	if b.Model.ListView.GetQuerySet != nil {
		qs = b.Model.ListView.GetQuerySet(AdminSite, b.Model.App(), b.Model)
	} else {
		qs = queries.GetQuerySet(b.Model.NewInstance())
	}
	if len(b.Model.ListView.Prefetch.SelectRelated) > 0 {
		qs = qs.SelectRelated(b.Model.ListView.Prefetch.SelectRelated...)
	}
	if len(b.Model.ListView.Prefetch.PrefetchRelated) > 0 {
		qs = qs.Preload(b.Model.ListView.Prefetch.PrefetchRelated...)
	}

	var orExprs = make([]expr.Expression, 0, len(b.Model.ListView.Search.Fields))
	for _, field := range b.Model.ListView.Search.Fields {

		var expression = field.AsExpression(req, searchQuery)
		if expression == nil {
			continue
		}

		orExprs = append(
			orExprs,
			expression,
		)
	}

	if len(orExprs) > 0 {
		qs = qs.Filter(expr.Or(orExprs...))
	}

	var (
		pageValue = req.URL.Query().Get("page")
		page      uint64
		err       error
	)

	if pageValue == "" {
		page = 1
	} else {
		page, err = strconv.ParseUint(pageValue, 10, 64)
	}
	if err != nil {
		return nil, err
	}

	var paginator = &pagination.QueryPaginator[attrs.Definer]{
		Context: req.Context(),
		Amount:  int(amount),
		BaseQuerySet: func() *queries.QuerySet[attrs.Definer] {
			return qs
		},
	}

	pageObject, err := paginator.Page(int(page))
	if err != nil && !errors.Is(err, errors.NoRows) {
		return nil, err
	}

	var results []attrs.Definer
	if pageObject != nil {
		results = pageObject.Results()
	}

	if !b.Model.DisallowEdit && len(results) > 0 {
		columns[0] = list.TitleFieldColumn(
			columns[0], func(r *http.Request, defs attrs.Definitions, row attrs.Definer) string {
				return b.GetEditLink(attrs.PrimaryKey(row))
			},
		)
	}

	listObj, err := b.GetList(b, results, columns)
	if err != nil {
		return nil, err
	}

	var contextPage = PageOptions{
		TitleFn: trans.S(
			"Search %s", b.Model.PluralLabel(req.Context()),
		),
	}

	if m, ok := listObj.(media.MediaDefiner); ok {
		contextPage.MediaFn = m.Media
	}

	var app = b.Model.App()

	context.SetPage(contextPage)
	context.Set("view_list", listObj)
	context.Set("view_page", page)
	context.Set("view_paginator", paginator)
	context.Set("view_paginator_object", pageObject)

	context.Set("view", b)
	context.Set("app", app)
	context.Set("model", b.Model)
	context.Set("query", searchQuery)
	context.Set("opts", b.Model.ListView.Search)
	context.Set("searchable_models", AdminSite.SearchableModels(req))

	return context, nil
}

func (b *BoundSearchView) GetEditLink(id any) string {
	var base string
	if b.Model.ListView.Search.GetEditLink != nil {
		base = b.Model.ListView.Search.GetEditLink(b.R, id)
	} else {
		base = django.Reverse(
			"admin:apps:model:edit",
			b.Model.App().Name,
			b.Model.GetName(),
			attrs.ToString(id),
		)
	}

	var sb = new(strings.Builder)
	sb.WriteString(base)

	if len(base) > 0 && base[len(base)-1] != '?' {
		sb.WriteString("?")
	}

	sb.WriteString("next=")
	sb.WriteString(url.QueryEscape(django.Reverse("admin:search")))
	sb.WriteString("%3F")
	sb.WriteString(url.QueryEscape(b.R.URL.RawQuery))
	return sb.String()
}
