package orders

import (
	queries "github.com/Nigel2392/go-django/queries/src"
)

type OrderQuery struct {
	*queries.WrappedQuerySet[*Order, *OrderQuery, *queries.QuerySet[*Order]]
}

func Orders() *OrderQuery {
	productQuerySet := &OrderQuery{}
	productQuerySet.WrappedQuerySet = queries.WrapQuerySet[*Order](
		queries.GetQuerySet(&Order{}),
		productQuerySet,
	)
	return productQuerySet
}

func (qs *OrderQuery) CloneQuerySet(wrapped *queries.WrappedQuerySet[*Order, *OrderQuery, *queries.QuerySet[*Order]]) *OrderQuery {
	return &OrderQuery{
		WrappedQuerySet: wrapped,
	}
}
