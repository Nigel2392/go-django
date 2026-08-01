package search

import (
	"context"
	"slices"

	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/attrutils"
)

type SearchField interface {
	Weight() int8   // higher is more important
	Field() string  // <fieldname> or <path.to.related.fieldname>
	Lookup() string // icontains, istartswith, exact, etc. (see expr.LookupFilter)
}

type SearchableModel interface {
	attrs.Definer
	SearchableFields() []SearchField
}

type Searchable interface {
	BuildSearchQuery(fields []BuiltSearchField, query string) (Searchable, error)
}

type SearchBackend interface {
	AddToSearchIndex(ctx context.Context, models []any) (int64, error)
	RemoveFromSearchIndex(ctx context.Context, models []any) (int64, error)
	Search(fields []BuiltSearchField, query string, searchable any) (any, error)
}

type BackendDefiner interface {
	SearchBackend() SearchBackend
}

func Search[MODEL SearchableModel, SEARCHABLE any](model MODEL, query string, searchable SEARCHABLE) (SEARCHABLE, error) {
	var backend, err = GetSearchBackendForModel(model)
	if err != nil {
		return *new(SEARCHABLE), err
	}

	fields, err := BuildSearchFields(model)
	if err != nil {
		return *new(SEARCHABLE), err
	}

	slices.SortStableFunc(fields, func(a, b BuiltSearchField) int {
		var (
			wa, wb = a.SearchField.Weight(), b.SearchField.Weight()
		)
		if wa != wb {
			return int(wa - wb)
		}
		return 0
	})

	b, err := backend.Search(fields, query, searchable)
	if err != nil {
		return *new(SEARCHABLE), err
	}

	if bld, ok := b.(Searchable); ok {
		bld, err = bld.BuildSearchQuery(fields, query)
		if err != nil {
			return *new(SEARCHABLE), err
		}
		return bld.(SEARCHABLE), nil
	}

	return b.(SEARCHABLE), nil
}

func IndexModels[T SearchableModel](ctx context.Context, models []T) (int64, error) {
	if len(models) == 0 {
		return 0, nil
	}

	var backend, err = GetSearchBackendForModel(*new(T))
	if err != nil {
		return 0, err
	}

	return backend.AddToSearchIndex(ctx, attrutils.InterfaceList(models))
}

func RemoveModelsFromIndex[T SearchableModel](ctx context.Context, models []T) (int64, error) {
	if len(models) == 0 {
		return 0, nil
	}

	var backend, err = GetSearchBackendForModel(*new(T))
	if err != nil {
		return 0, err
	}

	return backend.RemoveFromSearchIndex(ctx, attrutils.InterfaceList(models))
}
