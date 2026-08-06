package searches

import (
	"context"
	"net/http"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/views/list"
)

type SearchOptions struct {
	Order          int
	PerPage        int
	Fields         []SearchField
	ListFields     []string
	Model          attrs.Definer
	Prefetch       Prefetch
	PluralLabel    func(ctx context.Context) string
	GetEditLink    func(req *http.Request, id any) string
	Searchable     func(req *http.Request) bool
	AllowEdit      func(req *http.Request) bool
	QuerySet       func(req *http.Request) *queries.QuerySet[attrs.Definer]
	SearchQuerySet func(req *http.Request, qs *queries.QuerySet[attrs.Definer], query string) *queries.QuerySet[attrs.Definer]
	GetList        func(b *BoundSearchView, list []attrs.Definer, columns []list.ListColumn[attrs.Definer]) (list.StringRenderer, error)
	GetColumn      func(ctx context.Context, opts *SearchOptions, field string) list.ListColumn[attrs.Definer]
	ctype          contenttypes.GenericContentType
}

func (so *SearchOptions) CanSearch(r *http.Request) bool {
	if so == nil {
		return false
	}
	if so.Searchable == nil {
		return true
	}
	return (len(so.Fields) > 0 || so.SearchQuerySet != nil) && so.Searchable(r)
}

func (so *SearchOptions) ContentType() contenttypes.GenericContentType {
	return so.ctype
}
