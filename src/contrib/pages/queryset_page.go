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
	// _ queries.NullQuerySet[Page, *SpecificPageQuerySet]                         = (*SpecificPageQuerySet)(nil)
)

type specificPage struct {
	ids   []int64
	pages map[int64]*queries.Row[Page]
}

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
	specific.WrappedQuerySet = queries.WrapQuerySet[*PageNode](
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
	qs.WrappedQuerySet.NullQuerySet = qs.WrappedQuerySet.Base().StatusFlags(statusFlags)
	return qs
}

func (qs *SpecificPageQuerySet) Published() *SpecificPageQuerySet {
	qs = qs.Clone()
	qs.WrappedQuerySet.NullQuerySet = qs.WrappedQuerySet.Base().Published()
	return qs
}

func (qs *SpecificPageQuerySet) Unpublished() *SpecificPageQuerySet {
	qs = qs.Clone()
	qs.WrappedQuerySet.NullQuerySet = qs.WrappedQuerySet.Base().Unpublished()
	return qs
}

func (qs *SpecificPageQuerySet) Types(types ...any) *SpecificPageQuerySet {
	qs = qs.Clone()
	qs.WrappedQuerySet.NullQuerySet = qs.WrappedQuerySet.Base().Types(types...)
	return qs
}

func (qs *SpecificPageQuerySet) AncestorOf(node *PageNode, inclusive ...bool) *SpecificPageQuerySet {
	qs = qs.Clone()
	qs.WrappedQuerySet.NullQuerySet = qs.WrappedQuerySet.Base().AncestorOf(node, inclusive...)
	return qs
}

func (qs *SpecificPageQuerySet) DescendantOf(node *PageNode, inclusive ...bool) *SpecificPageQuerySet {
	qs = qs.Clone()
	qs.WrappedQuerySet.NullQuerySet = qs.WrappedQuerySet.Base().DescendantOf(node, inclusive...)
	return qs
}

func (qs *SpecificPageQuerySet) ChildrenOf(node *PageNode) *SpecificPageQuerySet {
	qs = qs.Clone()
	qs.WrappedQuerySet.NullQuerySet = qs.WrappedQuerySet.Base().ChildrenOf(node)
	return qs
}

func (qs *SpecificPageQuerySet) SiblingsOf(node *PageNode, inclusive ...bool) *SpecificPageQuerySet {
	qs = qs.Clone()
	qs.WrappedQuerySet.NullQuerySet = qs.WrappedQuerySet.Base().SiblingsOf(node, inclusive...)
	return qs
}

func (qs *SpecificPageQuerySet) Search(query string) *SpecificPageQuerySet {
	qs = qs.Clone()
	qs.WrappedQuerySet.NullQuerySet = qs.WrappedQuerySet.Base().Search(query)
	return qs
}

func (qs *SpecificPageQuerySet) All() (queries.Rows[Page], error) {
	// Use iter method of the base queryset to not have to
	// iterate over all rows multiple times.
	var rowCount, baseRows, err = qs.WrappedQuerySet.Base().Base().IterAll()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get all rows for specific pages")
	}

	// Create a new map to hold the specific page content type and ID-list pairs.
	var preloadMap = orderedmap.NewOrderedMap[string, *specificPage]()
	var rows = make(queries.Rows[Page], rowCount)
	var rowIdx int
	for row, err := range baseRows {
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get row %d for specific pages", rowIdx)
		}

		var specificPageRow = &queries.Row[Page]{
			ObjectFieldDefs: row.ObjectFieldDefs,
			Through:         row.Through,
			Annotations:     row.Annotations,

			// currently we set the object to the node isntead of the specific page,
			// if a specific page can be found, it will be properly set to the right object
			// later in the prefetch loop.
			Object: row.Object,
		}

		// If the page has no content type or page ID, we skip it,
		// it cannot be preloaded - log a warning.
		if row.Object.PageID == 0 || row.Object.ContentType == "" {
			logger.Warnf("page with ID %d has no content type or page ID, skipping preload", row.Object.PageID)

			// add the row to the results, increase idx
			rows[rowIdx] = specificPageRow
			rowIdx++
			continue
		}

		// Cache a reference to the specific page row in
		// the preload map - this allows us to efficiently
		// set the page object later in a single query
		// for each content type and id list combination.
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

		// add the row to the results, increase idx
		rows[rowIdx] = specificPageRow
		rowIdx++
	}

	if preloadMap.Len() == 0 {
		return rows, err
	}

	// prefetch all rows for each content type
	// and set the page object for each row.
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
			WithContext(qs.Context()).
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

// AddRoot adds a root page to the queryset.
//
// It is a wrapper around AddRoots that takes a single page.
func (qs *SpecificPageQuerySet) AddRoot(page Page) error {
	return qs.AddRoots(page)
}

// AddRoots adds multiple root pages to the queryset.
//
// It does so by creating PageNode objects for each page,
// and then passing it to the [PageQuerySet]'s AddRoots method.
func (qs *SpecificPageQuerySet) AddRoots(pages ...Page) error {
	var nodes = make([]*PageNode, 0, len(pages))
	for _, page := range pages {
		if page == nil {
			continue
		}

		var node = page.Reference()
		node.PageObject = page
		nodes = append(nodes, node)
	}

	return qs.WrappedQuerySet.Base().AddRoots(nodes...)
}

// AddChildren adds child pages to a parent page.
// It creates PageNode objects for each child page and
// then passes them to the [PageQuerySet]'s AddChildren method.
func (qs *SpecificPageQuerySet) AddChildren(parent Page, children ...Page) error {
	var parentNode = parent.Reference()
	parentNode.PageObject = parent

	var childNodes = make([]*PageNode, 0, len(children))
	for _, child := range children {
		if child == nil {
			continue
		}

		var childNode = child.Reference()
		childNode.PageObject = child
	}

	return qs.WrappedQuerySet.Base().AddChildren(parentNode, childNodes...)
}

// DeletePage deletes a page from the queryset.
func (qs *SpecificPageQuerySet) DeletePage(page Page) error {
	var node = page.Reference()
	node.PageObject = page
	_, err := qs.WrappedQuerySet.Base().Delete(node)
	return err
}

// MovePage moves a page and all it's children under a new parent page.
func (qs *SpecificPageQuerySet) MovePage(page Page, newParent Page) error {
	var (
		ref1 = page.Reference()
		ref2 = newParent.Reference()
	)
	ref1.PageObject = page
	ref2.PageObject = newParent
	return qs.WrappedQuerySet.Base().MoveNode(ref1, ref2)
}

// PublishPage publishes the given page.
func (qs *SpecificPageQuerySet) PublishPage(page Page) error {
	var node = page.Reference()
	node.PageObject = page
	return qs.WrappedQuerySet.Base().PublishNode(node)
}

// UnpublishPage unpublishes the given page.
func (qs *SpecificPageQuerySet) UnpublishPage(page Page, unpublishChildren bool) error {
	var node = page.Reference()
	node.PageObject = page
	return qs.WrappedQuerySet.Base().UnpublishNode(node, unpublishChildren)
}
