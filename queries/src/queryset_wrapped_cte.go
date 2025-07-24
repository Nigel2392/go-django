package queries

import (
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/elliotchance/orderedmap/v2"
)

type CTE[T attrs.Definer] struct {
	Name     string
	QuerySet QuerySet[T]
	fields   comparingField
}

type CTEQuerySet[T attrs.Definer] struct {
	*WrappedQuerySet[T, *CTEQuerySet[T], *QuerySet[T]]
	CTE *orderedmap.OrderedMap[string, *CTE[T]]
}
