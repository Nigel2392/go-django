package admin

import (
	"net/http"
	"strings"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/views/list"
)

type SearchOptions struct {
	PerPage        int
	Fields         []SearchField
	ListFields     []string
	GetEditLink    func(req *http.Request, id any) string
	Searchable     func(req *http.Request) bool
	QuerySet       func(req *http.Request) *queries.QuerySet[attrs.Definer]
	SearchQuerySet func(req *http.Request, qs *queries.QuerySet[attrs.Definer], query string) *queries.QuerySet[attrs.Definer]
	GetList        func(b *BoundSearchView, list []attrs.Definer, columns []list.ListColumn[attrs.Definer]) (StringRenderer, error)
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

type SearchField struct {
	Name            string
	Lookup          string // expr.LOOKUP_EXACT is the default.
	BuildExpression func(sf SearchField, value any) expr.Expression
}

func NewSearchField(nameWithlookup string) SearchField {
	var parts = strings.SplitN(nameWithlookup, "__", 2)
	var name = parts[0]
	var lookup = expr.LOOKUP_EXACT
	if len(parts) > 1 {
		lookup = parts[1]
	}
	return SearchField{
		Name:   name,
		Lookup: lookup,
	}
}

func (sf SearchField) FieldName() string {
	return sf.Name
}

func (sf SearchField) FilterName() string {
	if sf.Lookup == "" {
		return sf.Name
	}
	var b = make([]byte, 0, len(sf.Name)+len(sf.Lookup)+2)
	b = append(b, sf.Name...)
	b = append(b, '_')
	b = append(b, '_')
	b = append(b, sf.Lookup...)
	return string(b)
}

func (sf SearchField) AsExpression(value any) expr.Expression {
	if sf.BuildExpression != nil {
		return sf.BuildExpression(sf, value)
	}

	return expr.Q(sf.FilterName(), value)
}
