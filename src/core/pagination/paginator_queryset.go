package pagination

import (
	"context"
	"reflect"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

// QueryPaginator holds pagination logic
type QueryPaginator[T attrs.Definer] struct {
	Context      context.Context
	BaseQuerySet func() *queries.QuerySet[T]
	GetObject    func(T) T
	URL          string
	Amount       int
	cnt          int
}

func (p *QueryPaginator[T]) NumPages() (int, error) {
	count, err := p.Count()
	if err != nil {
		return 0, err
	}
	return (count + p.PerPage() - 1) / p.PerPage(), nil
}

func (p *QueryPaginator[T]) GetQuerySet() *queries.QuerySet[T] {
	var qs *queries.QuerySet[T]
	if p.BaseQuerySet == nil {
		qs = queries.GetQuerySet[T](attrs.NewObject[T](
			reflect.TypeOf(new(T)).Elem(),
		))
	} else {
		qs = p.BaseQuerySet()
	}
	return qs.WithContext(p.Context)
}

func (p *QueryPaginator[T]) Count() (int, error) {
	if p.cnt == 0 {
		count, err := p.GetQuerySet().Count()
		if err != nil {
			return 0, err
		}
		p.cnt = int(count)
	}
	return p.cnt, nil
}

func (p *QueryPaginator[T]) BaseURL() string {
	return p.URL
}

func (p *QueryPaginator[T]) PerPage() int {
	return p.Amount
}

func (p *QueryPaginator[T]) Page(n int) (PageObject[T], error) {
	var amount = p.PerPage()

	if amount == 0 {
		return nullPageObject(p.Context, p), errors.ValueError.Wrapf(
			"amount of objects per page cannot be 0",
		)
	}

	var offset = (n - 1) * amount
	var resultCnt, resultIter, err = p.GetQuerySet().
		Offset(offset).
		Limit(amount).
		IterAll()
	if err != nil {
		return nullPageObject(p.Context, p), err
	}

	if resultCnt == 0 {
		return nullPageObject(p.Context, p), errors.NoRows
	}

	var idx = 0
	var resultRows = make([]T, resultCnt)
	for row, err := range resultIter {
		if err != nil {
			return nullPageObject(p.Context, p), errors.Wrapf(err, "failed to get row %d", idx)
		}

		var obj = row.Object
		if p.GetObject != nil {
			obj = p.GetObject(row.Object)
		}

		resultRows[idx] = obj
		idx++
	}

	return NewPageObject(p.Context, p, n, resultRows), nil
}
