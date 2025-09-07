package searchqueries

import (
	"context"
	"fmt"

	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/contrib/search"
	"github.com/Nigel2392/go-django/src/core/attrs/attrutils"
)

func init() {
	search.RegisterSearchBackend("queries", &QueriesSearchBackend{}, true)
}

var _ search.SearchBackend = (*QueriesSearchBackend)(nil)

type QueriesSearchBackend struct{}

func (q *QueriesSearchBackend) AddToSearchIndex(ctx context.Context, models []any) (int64, error) {
	return 0, nil
}

func (q *QueriesSearchBackend) RemoveFromSearchIndex(ctx context.Context, models []any) (int64, error) {
	return 0, nil
}

func (q *QueriesSearchBackend) Search(fields []search.BuiltSearchField, query string, searchable any) (any, error) {
	var cases = make([]expr.WhenExpression, 0, len(fields))
	var orexprs = make([]expr.Expression, 0, len(fields))
	for _, sf := range fields {

		var fieldName = sf.SearchField.Field()
		var lookup = sf.SearchField.Lookup()
		if lookup == "" {
			lookup = expr.LOOKUP_IEXACT
		}

		cases = append(cases, expr.When(
			fmt.Sprintf("%s__%s", fieldName, lookup), query).
			Then(sf.SearchField.Weight()),
		)

		orexprs = append(orexprs, expr.Expr(
			fieldName, lookup, query,
		))
	}

	var err error
	var caseExpr = expr.Case(cases)
	var bld = attrutils.NewBuilder(searchable)
	searchable, _, err = bld.Chain("Annotate", "_relevance", caseExpr).
		Chain("Filter", expr.Or(orexprs...)).
		Call("OrderBy", "-_relevance")

	if err != nil {
		return nil, err
	}

	return searchable, nil
}
