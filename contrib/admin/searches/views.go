package searches

import (
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/Nigel2392/go-django/contrib/messages"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/pagination"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/go-django/src/permissions"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/go-django/src/views/list"
)

var (
	_ views.View         = &SearchView{}
	_ views.BindableView = &SearchView{}
)

type SearchView struct {
	views.BaseView
	FailURL          func(r *http.Request) string
	GetSearchOption  func(r *http.Request, cTypeName string) *SearchOptions
	GetSearchOptions func(r *http.Request) []*SearchOptions
	GetContext       func(r *http.Request, v *BoundSearchView) ctx.ContextWithRequest
	SetupOptions     func(r *http.Request, opts *SearchOptions)
}

func (s *SearchView) setupOptions(r *http.Request, opts *SearchOptions) {

	if opts.ctype != nil {
		return
	}

	opts.ctype = contenttypes.NewContentType(opts.Model)

	if opts.PluralLabel == nil {
		opts.PluralLabel = contenttypes.
			DefinitionForObject(opts.Model).
			PluralLabel
	}

	if s.SetupOptions != nil {
		s.SetupOptions(r, opts)
	}
}

func (s *SearchView) getSearchOptions(r *http.Request) []*SearchOptions {
	var opts = s.GetSearchOptions(r)
	for _, opt := range opts {
		s.setupOptions(r, opt)
	}
	return opts
}

func (s *SearchView) ServeXXX(w http.ResponseWriter, req *http.Request) {
}

func (s *SearchView) Methods() []string {
	return []string{http.MethodGet}
}

func (s *SearchView) Bind(w http.ResponseWriter, req *http.Request) (views.View, error) {

	if s.GetSearchOption == nil || s.GetSearchOptions == nil {
		return nil, except.Fail(
			http.StatusInternalServerError,
			"Improperly configured.",
		)
	}

	if s.FailURL == nil {
		s.FailURL = func(r *http.Request) string { return "" }
	}

	var v = &BoundSearchView{SearchView: s, BaseView: s.BaseView}
	return v, nil
}

type BoundSearchView struct {
	views.BaseView
	SearchView *SearchView
	W          http.ResponseWriter
	R          *http.Request
	Search     *SearchOptions
	Searches   []*SearchOptions
}

func (b *BoundSearchView) Setup(w http.ResponseWriter, req *http.Request) (http.ResponseWriter, *http.Request) {
	b.W = w
	b.R = req
	b.Searches = b.SearchView.getSearchOptions(req)

	var cTypeName = req.URL.Query().Get("content_type")
	if cTypeName != "" {
		b.Search = b.SearchView.GetSearchOption(req, cTypeName)
		b.SearchView.setupOptions(req, b.Search)
	}

	slices.SortStableFunc(b.Searches, func(a, b *SearchOptions) int {
		return a.Order - b.Order
	})

	if b.Search == nil {
		if len(b.Searches) == 0 {
			except.Fail(
				http.StatusNotFound,
				"no searchable models found in registry",
				nil,
			)
			return nil, nil
		}

		b.Search = b.Searches[0]
	}

	if b.Search == nil || !b.Search.CanSearch(req) {
		messages.Error(
			req, trans.T(req.Context(), "Search is not allowed for this model"),
		)
		http.Redirect(
			w,
			req,
			b.SearchView.FailURL(req),
			http.StatusFound,
		)
		return nil, nil
	}

	if b.Search.Model == nil {
		messages.Error(
			req, trans.T(req.Context(), "You must search a model."),
		)
		http.Redirect(
			w,
			req,
			b.SearchView.FailURL(req),
			http.StatusFound,
		)
		return nil, nil
	}

	return w, req
}

func (b *BoundSearchView) GetList(v *BoundSearchView, objects []attrs.Definer, columns []list.ListColumn[attrs.Definer]) (list.StringRenderer, error) {
	if b.Search.GetList != nil {
		return b.Search.GetList(b, objects, columns)
	}
	return list.NewList(v.R, attrs.NewObject[attrs.Definer](v.R.Context(), b.Search.Model), objects, columns...), nil
}

func (b *BoundSearchView) GetContext(req *http.Request) (ctx.Context, error) {
	var (
		fields      = b.Search.ListFields
		context     = b.SearchView.GetContext(req, b)
		searchQuery = req.URL.Query().Get("global-search")
		columns     = make([]list.ListColumn[attrs.Definer], len(fields))
	)
	for i, field := range fields {
		columns[i] = b.Search.GetColumn(
			req.Context(), b.Search, field,
		)
	}

	var amount = b.Search.PerPage
	if amount == 0 {
		amount = 25
	}

	var qs *queries.QuerySet[attrs.Definer]
	switch {
	case b.Search.QuerySet != nil:
		qs = b.Search.QuerySet(req)
	default:
		qs = queries.GetQuerySet(b.Search.Model)
	}

	qs = qs.WithContext(req.Context())

	if len(b.Search.Prefetch.SelectRelated) > 0 {
		qs = qs.SelectRelated(b.Search.Prefetch.SelectRelated...)
	}
	if len(b.Search.Prefetch.PrefetchRelated) > 0 {
		qs = qs.Preload(b.Search.Prefetch.PrefetchRelated...)
	}
	if b.Search.SearchQuerySet != nil {
		qs = b.Search.SearchQuerySet(req, qs, searchQuery)
	}

	var orExprs = make([]expr.Expression, 0, len(b.Search.Fields))
	for _, field := range b.Search.Fields {

		var expression = field.AsExpression(searchQuery)
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

	if (b.Search.AllowEdit(req) || b.Search.GetEditLink != nil) && len(results) > 0 && permissions.HasPermission(req, "admin:edit") {
		columns[0] = list.TitleFieldColumn(
			columns[0], func(r *http.Request, defs attrs.Definitions, row attrs.Definer) string {
				return b.GetEditLink(attrs.PrimaryKey(req.Context(), row))
			},
		)
	}

	listObj, err := b.GetList(b, results, columns)
	if err != nil {
		return nil, err
	}

	if m, ok := listObj.(media.MediaDefiner); ok {
		context.Set("media", m.Media())
	}

	context.Set("view_list", listObj)
	context.Set("view_page", page)
	context.Set("view_paginator", paginator)
	context.Set("view_paginator_object", pageObject)

	context.Set("view", b)
	context.Set("query", searchQuery)
	context.Set("opts", b.Search)
	context.Set("searchable_models", b.Searches)
	context.Set("ctype", b.Search.ctype.ShortTypeName())

	return context, nil
}

func (b *BoundSearchView) GetEditLink(id any) string {
	var base = b.Search.GetEditLink(b.R, id)
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
