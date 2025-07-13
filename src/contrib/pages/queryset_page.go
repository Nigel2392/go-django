package pages

import (
	"fmt"
	"strconv"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/elliotchance/orderedmap/v2"
)

var (
	// _ queries.QuerySetCanBeforeExec = (*PageQuerySet)(nil)
	// _ queries.QuerySetCanAfterExec                                              = (*SpecificPageQuerySet)(nil)
	_ queries.QuerySetCanClone[*PageNode, *SpecificPageQuerySet, *PageQuerySet] = (*SpecificPageQuerySet)(nil)
	// _ queries.BaseQuerySet[Page, *SpecificPageQuerySet]                         = (*SpecificPageQuerySet)(nil)
)

type specificPage struct {
	ids   []int64
	pages map[int64]*queries.Row[Page]
}

type specificPreloadInfo = orderedmap.OrderedMap[string, *specificPage]

func variableBool(b ...bool) bool {
	var v bool
	if len(b) > 0 {
		v = b[0]
	}
	return v
}

type SpecificPageQuerySet struct {
	*queries.WrappedQuerySet[*PageNode, *SpecificPageQuerySet, *PageQuerySet]
}

func newSpecificPageQuerySet(qs *PageQuerySet) *SpecificPageQuerySet {
	var specific = &SpecificPageQuerySet{}
	specific.WrappedQuerySet = queries.WrapQuerySet(
		qs, specific,
	)
	return specific
}

func (qs *SpecificPageQuerySet) CloneQuerySet(wrapped *queries.WrappedQuerySet[*PageNode, *SpecificPageQuerySet, *PageQuerySet]) *SpecificPageQuerySet {
	return &SpecificPageQuerySet{
		WrappedQuerySet: wrapped,
	}
}

func (qs *SpecificPageQuerySet) StatusFlags(statusFlags StatusFlag) *SpecificPageQuerySet {
	qs = qs.Clone()
	qs.WrappedQuerySet.BaseQuerySet = qs.WrappedQuerySet.Base().StatusFlags(statusFlags)
	return qs
}

func (qs *SpecificPageQuerySet) Published() *SpecificPageQuerySet {
	qs = qs.Clone()
	qs.WrappedQuerySet.BaseQuerySet = qs.WrappedQuerySet.Base().Published()
	return qs
}

func (qs *SpecificPageQuerySet) Unpublished() *SpecificPageQuerySet {
	qs = qs.Clone()
	qs.WrappedQuerySet.BaseQuerySet = qs.WrappedQuerySet.Base().Unpublished()
	return qs
}

func (qs *SpecificPageQuerySet) Types(types ...any) *SpecificPageQuerySet {
	qs = qs.Clone()
	qs.WrappedQuerySet.BaseQuerySet = qs.WrappedQuerySet.Base().Types()
	return qs
}

func (qs *SpecificPageQuerySet) AncestorOf(node *PageNode, inclusive ...bool) *SpecificPageQuerySet {
	qs = qs.Clone()
	qs.WrappedQuerySet.BaseQuerySet = qs.WrappedQuerySet.Base().AncestorOf(node, inclusive...)
	return qs
}

func (qs *SpecificPageQuerySet) DescendantOf(node *PageNode, inclusive ...bool) *SpecificPageQuerySet {
	qs = qs.Clone()
	qs.WrappedQuerySet.BaseQuerySet = qs.WrappedQuerySet.Base().DescendantOf(node, inclusive...)
	return qs
}

func (qs *SpecificPageQuerySet) ChildrenOf(node *PageNode) *SpecificPageQuerySet {
	qs = qs.Clone()
	qs.WrappedQuerySet.BaseQuerySet = qs.WrappedQuerySet.Base().ChildrenOf(node)
	return qs
}

func (qs *SpecificPageQuerySet) SiblingsOf(node *PageNode, inclusive ...bool) *SpecificPageQuerySet {
	qs = qs.Clone()
	qs.WrappedQuerySet.BaseQuerySet = qs.WrappedQuerySet.Base().SiblingsOf(node, inclusive...)
	return qs
}

func (qs *SpecificPageQuerySet) All() (queries.Rows[Page], error) {
	var rowCount, baseRows, err = qs.WrappedQuerySet.Base().Base().IterAll()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get all rows for specific pages")
	}

	var rowIdx int
	var rows = make(queries.Rows[Page], rowCount)
	var preloadMap = orderedmap.NewOrderedMap[string, *specificPage]()
	for row, err := range baseRows {
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get row %d for specific pages", rowIdx)
		}

		var specificPageRow = &queries.Row[Page]{
			Object:          row.Object.PageObject,
			ObjectFieldDefs: row.ObjectFieldDefs,
			Through:         row.Through,
			Annotations:     row.Annotations,
		}

		// If the page has no content type or page ID, we skip it,
		// it cannot be preloaded - we will add the page node as is.
		if row.Object.PageID == 0 || row.Object.ContentType == "" {
			logger.Warnf("page with ID %d has no content type or page ID, skipping preload", row.Object.PageID)
			specificPageRow.Object = row.Object
			rows[rowIdx] = specificPageRow
			rowIdx++
			continue
		}

		var preload, exists = preloadMap.Get(
			row.Object.ContentType,
		)
		if !exists {
			preload = &specificPage{
				ids:   make([]int64, 0, 1),
				pages: make(map[int64]*queries.Row[Page], 1),
			}
		}

		preload.ids = append(preload.ids, row.Object.PageID)
		preload.pages[row.Object.PageID] = specificPageRow

		preloadMap.Set(
			row.Object.ContentType,
			preload,
		)

		rows[rowIdx] = specificPageRow
		rowIdx++
	}

	if preloadMap.Len() == 0 {
		return rows, err
	}

	// Reset the preload map

	for head := preloadMap.Front(); head != nil; head = head.Next() {
		var definition = DefinitionForType(head.Key)
		if definition == nil {
			return rows, errors.New(errors.CodeNoRows, fmt.Sprintf(
				"no content type definition found for %s",
				head.Key,
			))
		}

		var model = definition.Object().(Page)
		var defs = model.FieldDefs()
		var primaryField = defs.Primary()
		var rows, err = queries.GetQuerySet(model).
			Filter(fmt.Sprintf("%s__in", primaryField.Name()), head.Value.ids).
			All()
		if err != nil {
			return rows, errors.Wrapf(err, "failed to get rows for content type %s", head.Key)
		}

		for _, row := range rows {
			var primary = row.ObjectFieldDefs.Primary()
			var pk = attrs.Get[int64](row.ObjectFieldDefs, primary.Name())
			var pageRow, exists = head.Value.pages[pk]
			if !exists {
				return nil, errors.New(errors.CodeNoRows, fmt.Sprintf(
					"no page found for content type %s with PK %d",
					head.Key, pk,
				))
			}

			pageRow.Object = row.Object
		}
	}

	return rows, err
}

func (qs *SpecificPageQuerySet) Get() (*queries.Row[Page], error) {
	var nillRow = &queries.Row[Page]{}

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
func (qs *SpecificPageQuerySet) First() (*queries.Row[Page], error) {
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
func (qs *SpecificPageQuerySet) Last() (*queries.Row[Page], error) {
	*qs = *qs.Reverse()
	return qs.First()
}
