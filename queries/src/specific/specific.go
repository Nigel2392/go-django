package specific

import (
	"fmt"
	"iter"
	"strconv"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/elliotchance/orderedmap/v2"
)

var (
	_ queries.QuerySetCanClone[attrs.Definer, *SpecificQuerySet[attrs.Definer, attrs.Definer, *queries.QuerySet[attrs.Definer]], *queries.QuerySet[attrs.Definer]] = (*SpecificQuerySet[attrs.Definer, attrs.Definer, *queries.QuerySet[attrs.Definer]])(nil)
	_ queries.BaseReadQuerySet[Specific[attrs.Definer, attrs.Definer], *SpecificQuerySet[attrs.Definer, attrs.Definer, *queries.QuerySet[attrs.Definer]]]          = (*SpecificQuerySet[attrs.Definer, attrs.Definer, *queries.QuerySet[attrs.Definer]])(nil)
	_ queries.BaseWriteQuerySet[attrs.Definer, *SpecificQuerySet[attrs.Definer, attrs.Definer, *queries.QuerySet[attrs.Definer]]]                                  = (*SpecificQuerySet[attrs.Definer, attrs.Definer, *queries.QuerySet[attrs.Definer]])(nil)
)

type BaseSpecificQuerySet[ORIGINAL attrs.Definer, SPECIFIC attrs.Definer, QS queries.NullQuerySet[ORIGINAL, QS]] interface {
	queries.BaseReadQuerySet[Specific[ORIGINAL, SPECIFIC], *SpecificQuerySet[ORIGINAL, SPECIFIC, QS]]
	queries.BaseWriteQuerySet[ORIGINAL, *SpecificQuerySet[ORIGINAL, SPECIFIC, QS]]
}

type specific[ORIGINAL attrs.Definer, SPECIFIC attrs.Definer] struct {
	ids []any
	s   map[any]*queries.Row[Specific[ORIGINAL, SPECIFIC]]
}

type Specific[ORIGINAL attrs.Definer, SPECIFIC attrs.Definer] struct {
	Original ORIGINAL
	Specific SPECIFIC
}

type SpecificQuerySetOptions[ORIGINAL attrs.Definer, SPECIFIC attrs.Definer] struct {
	GetSpecificQuerySet    func(targetContentType *contenttypes.ContentTypeDefinition, target SPECIFIC) queries.BaseReadQuerySet[SPECIFIC, *queries.QuerySet[SPECIFIC]]
	GetSpecificPreloadData func(obj ORIGINAL) (id any, contentType string, ok bool)
	GetSpecificTargetID    func(target SPECIFIC) (id any, ok bool)
}

type SpecificQuerySet[ORIGINAL attrs.Definer, SPECIFIC attrs.Definer, QS queries.NullQuerySet[ORIGINAL, QS]] struct {
	*queries.WrappedQuerySet[ORIGINAL, *SpecificQuerySet[ORIGINAL, SPECIFIC, QS], QS]
	opts SpecificQuerySetOptions[ORIGINAL, SPECIFIC]
}

func GetSpecificQuerySet[ORIGINAL attrs.Definer, SPECIFIC attrs.Definer, QS queries.NullQuerySet[ORIGINAL, QS]](qs QS, opts SpecificQuerySetOptions[ORIGINAL, SPECIFIC]) *SpecificQuerySet[ORIGINAL, SPECIFIC, QS] {
	assert.False(
		opts.GetSpecificPreloadData == nil,
		"GetSpecificQuerySet: opts.GetSpecificPreloadData must be provided",
	)
	assert.False(
		opts.GetSpecificTargetID == nil,
		"GetSpecificQuerySet: opts.GetSpecificTargetID must be provided",
	)

	var specific = &SpecificQuerySet[ORIGINAL, SPECIFIC, QS]{
		opts: opts,
	}
	specific.WrappedQuerySet = queries.WrapQuerySet[ORIGINAL](
		qs, specific,
	)
	return specific
}

func (qs *SpecificQuerySet[ORIGINAL, SPECIFIC, QS]) CloneQuerySet(wrapped *queries.WrappedQuerySet[ORIGINAL, *SpecificQuerySet[ORIGINAL, SPECIFIC, QS], QS]) *SpecificQuerySet[ORIGINAL, SPECIFIC, QS] {
	return &SpecificQuerySet[ORIGINAL, SPECIFIC, QS]{
		WrappedQuerySet: wrapped,
		opts:            qs.opts,
	}
}

func (qs *SpecificQuerySet[ORIGINAL, SPECIFIC, QS]) IterAll() (int, iter.Seq2[*queries.Row[Specific[ORIGINAL, SPECIFIC]], error], error) {
	var count, rows, err = qs.WrappedQuerySet.IterAll()
	if err != nil {
		return 0, nil, err
	}

	var iterable = func(yield func(*queries.Row[Specific[ORIGINAL, SPECIFIC]], error) bool) {
		for row, err := range rows {
			if err != nil {
				if !yield(nil, err) {
					break
				}
			}

			var specificRow = &queries.Row[Specific[ORIGINAL, SPECIFIC]]{
				ObjectFieldDefs: row.ObjectFieldDefs,
				Through:         row.Through,
				Annotations:     row.Annotations,
				Object: Specific[ORIGINAL, SPECIFIC]{
					Original: row.Object,
				},
			}

			if !yield(specificRow, nil) {
				break
			}
		}
	}

	return count, iterable, nil
}

func (qs *SpecificQuerySet[ORIGINAL, SPECIFIC, QS]) All() (queries.Rows[Specific[ORIGINAL, SPECIFIC]], error) {
	// Use iter method of the base queryset to not have to
	// iterate over all rows multiple times.
	var rowCount, baseRows, err = qs.WrappedQuerySet.IterAll()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get all rows for specific s")
	}

	// Create a new map to hold the specific  content type and ID-list pairs.
	var preloadMap = orderedmap.NewOrderedMap[string, *specific[ORIGINAL, SPECIFIC]]()
	var rows = make(queries.Rows[Specific[ORIGINAL, SPECIFIC]], rowCount)
	var rowIdx int
	for row, err := range baseRows {
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get row %d for specific s", rowIdx)
		}

		var specificRow = &queries.Row[Specific[ORIGINAL, SPECIFIC]]{
			ObjectFieldDefs: row.ObjectFieldDefs,
			Through:         row.Through,
			Annotations:     row.Annotations,

			// currently we set the object to the node isntead of the specific ,
			// if a specific  can be found, it will be properly set to the right object
			// later in the prefetch loop.
			Object: Specific[ORIGINAL, SPECIFIC]{
				Original: row.Object,
			},
		}

		// If the object's SpecificPreloadData method returns ok as false,
		// it cannot be preloaded - log a warning.
		var preloadId, contentType, ok = qs.opts.GetSpecificPreloadData(row.Object)
		if !ok {
			logger.Warnf("object %T with preload ID %d cannot be preloaded", row.Object, preloadId)
			rows[rowIdx] = specificRow
			rowIdx++
			continue
		}

		// Cache a reference to the specific  row in
		// the preload map - this allows us to efficiently
		// set the  object later in a single query
		// for each content type and id list combination.
		var preload, exists = preloadMap.Get(
			contentType,
		)
		if !exists {
			preload = &specific[ORIGINAL, SPECIFIC]{
				ids: make([]any, 0, 1),
				s:   make(map[any]*queries.Row[Specific[ORIGINAL, SPECIFIC]], 1),
			}
		}

		preload.ids = append(preload.ids, preloadId)
		preload.s[preloadId] = specificRow

		preloadMap.Set(
			contentType,
			preload,
		)

		// add the row to the results, increase idx
		rows[rowIdx] = specificRow
		rowIdx++
	}

	if preloadMap.Len() == 0 {
		return rows, err
	}

	// prefetch all rows for each content type
	// and set the  object for each row.
	for head := preloadMap.Front(); head != nil; head = head.Next() {
		var definition = contenttypes.DefinitionForType(head.Key)
		if definition == nil {
			return rows, errors.New(errors.CodeNoRows, fmt.Sprintf(
				"no content type definition found for %s",
				head.Key,
			))
		}

		var (
			model        = definition.Object().(SPECIFIC)
			defs         = model.FieldDefs()
			primaryField = defs.Primary()
		)

		var baseQS queries.BaseReadQuerySet[SPECIFIC, *queries.QuerySet[SPECIFIC]]
		if qs.opts.GetSpecificQuerySet != nil {
			baseQS = qs.opts.GetSpecificQuerySet(definition, model)
		} else {
			baseQS = queries.GetQuerySet(model)
		}

		var rows, err = baseQS.
			WithContext(qs.Context()).
			Filter(fmt.Sprintf("%s__in", primaryField.Name()), head.Value.ids).
			All()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get specific rows for content type %s", head.Key)
		}

		for _, row := range rows {
			var pk, ok = qs.opts.GetSpecificTargetID(row.Object)
			if !ok {
				return nil, errors.New(errors.CodeNoRows, fmt.Sprintf(
					"no specific target ID found for content type %s",
					head.Key,
				))
			}

			var originalRow, exists = head.Value.s[pk]
			if !exists {
				return nil, errors.New(errors.CodeNoRows, fmt.Sprintf(
					"no original row found for content type %s with PK %d",
					head.Key, pk,
				))
			}

			originalRow.Object.Specific = row.Object
		}
	}

	return rows, err
}

func (qs *SpecificQuerySet[ORIGINAL, SPECIFIC, QS]) Get() (*queries.Row[Specific[ORIGINAL, SPECIFIC]], error) {
	var nillRow = &queries.Row[Specific[ORIGINAL, SPECIFIC]]{}

	// limit to max_get_results
	*qs = *qs.Limit(queries.MAX_GET_RESULTS)

	var results, err = qs.All()
	if err != nil {
		return nillRow, err
	}

	var resCnt = len(results)
	if resCnt == 0 {
		return nillRow, errors.NoRows.WithCause(fmt.Errorf(
			"no rows found for %T", qs.Meta().Model(),
		))
	}

	if resCnt > 1 {
		var errResCnt string
		if queries.MAX_GET_RESULTS == 0 || resCnt < queries.MAX_GET_RESULTS {
			errResCnt = strconv.Itoa(resCnt)
		} else {
			errResCnt = strconv.Itoa(queries.MAX_GET_RESULTS-1) + "+"
		}

		return nillRow, errors.MultipleRows.WithCause(fmt.Errorf(
			"multiple rows returned for %T: %s rows",
			qs.Meta().Model(), errResCnt,
		))
	}

	return results[0], nil
}

// First is used to retrieve the first row from the database.
//
// It returns a Query that can be executed to get the result, which is a Row object
// that contains the model object and a map of annotations.
func (qs *SpecificQuerySet[ORIGINAL, SPECIFIC, QS]) First() (*queries.Row[Specific[ORIGINAL, SPECIFIC]], error) {
	*qs = *qs.Limit(1)

	var results, err = qs.All()
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, errors.NoRows
	}

	return results[0], nil
}

// Last is used to retrieve the last row from the database.
//
// It reverses the order of the results and then calls First to get the last row.
//
// It returns a Query that can be executed to get the result, which is a Row object
// that contains the model object and a map of annotations.
func (qs *SpecificQuerySet[ORIGINAL, SPECIFIC, QS]) Last() (*queries.Row[Specific[ORIGINAL, SPECIFIC]], error) {
	*qs = *qs.Reverse()
	return qs.First()
}
