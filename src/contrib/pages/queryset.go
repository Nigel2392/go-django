package pages

import (
	"fmt"
	"strings"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/contrib/search"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/attrutils"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/elliotchance/orderedmap/v2"
)

var (
	// _ queries.QuerySetCanBeforeExec = (*PageQuerySet)(nil)
	_ queries.QuerySetCanClone[*PageNode, *PageQuerySet, *queries.QuerySet[*PageNode]] = (*PageQuerySet)(nil)
)

type PageQuerySet struct {
	*queries.WrappedQuerySet[*PageNode, *PageQuerySet, *queries.QuerySet[*PageNode]]
}

func NewPageQuerySet() *PageQuerySet {
	return wrapPageQuerySet(queries.GetQuerySet(&PageNode{}))
}

func wrapPageQuerySet(qs *queries.QuerySet[*PageNode]) *PageQuerySet {
	var pqs = &PageQuerySet{}
	pqs.WrappedQuerySet = queries.WrapQuerySet[*PageNode](qs, pqs)
	return pqs
}

func (qs *PageQuerySet) CloneQuerySet(wrapped *queries.WrappedQuerySet[*PageNode, *PageQuerySet, *queries.QuerySet[*PageNode]]) *PageQuerySet {
	return &PageQuerySet{
		WrappedQuerySet: wrapped,
	}
}

func (qs *PageQuerySet) Specific() *SpecificPageQuerySet {
	qs = qs.Clone()
	return newSpecificPageQuerySet(qs)
}

func (qs *PageQuerySet) StatusFlags(statusFlags StatusFlag) *PageQuerySet {
	return qs.Filter(qs.statusFlags(statusFlags))
}

func (qs *PageQuerySet) Published() *PageQuerySet {
	return qs.Filter(qs.statusFlags(StatusFlagPublished))
}

func (qs *PageQuerySet) Unpublished() *PageQuerySet {
	return qs.Filter(qs.statusFlags(StatusFlagPublished).Not(true))
}

func (qs *PageQuerySet) RootPages() *PageQuerySet {
	return qs.Filter("Depth", 0)
}

func (qs *PageQuerySet) Types(types ...any) *PageQuerySet {
	if len(types) == 0 {
		return qs
	}

	var typeNames = make([]string, len(types))
	for i, t := range types {
		switch v := t.(type) {
		case string:
			typeNames[i] = contenttypes.ReverseAlias(v)
		case attrs.Definer:
			typeNames[i] = contenttypes.NewContentType(v).TypeName()
		case contenttypes.ContentType:
			typeNames[i] = v.TypeName()
		default:
			panic(fmt.Errorf(
				"invalid type %T for ForTypes, expected string, attrs.Definer or contenttypes.ContentType", t,
			))
		}
	}

	if len(typeNames) == 1 {
		return qs.Filter("ContentType", typeNames[0])
	}

	return qs.Filter("ContentType__in", typeNames)
}

func (qs *PageQuerySet) Ancestors(path string, depth int64, inclusive ...bool) *PageQuerySet {
	depth++

	var incl = variableBool(inclusive...)
	var paths = make([]string, depth)
	var start = 0
	if !incl {
		start = 1
	}
	for i := start; i < int(depth); i++ {
		var path, err = ancestorPath(
			path, int64(i),
		)
		if err != nil {
			panic(errors.Wrapf(
				err, "failed to get ancestor path for %s at depth %d",
				path, i,
			))
		}
		paths[i] = path
	}

	return qs.Filter("Path__in", paths)
}

func (qs *PageQuerySet) Descendants(path string, depth int64, inclusive ...bool) *PageQuerySet {
	var incl = variableBool(inclusive...)
	var exp = expr.And(
		expr.Expr("Path", expr.LOOKUP_STARTSWITH, path),
		expr.Expr("Depth", expr.LOOKUP_GT, depth),
	)

	if incl {
		exp = expr.Or[any](
			exp,
			expr.Expr("Path", expr.LOOKUP_EXACT, path),
		)
	}

	return qs.Filter(exp)
}

func (qs *PageQuerySet) Children(path string, depth int64) *PageQuerySet {
	return qs.Filter(
		expr.Expr("Path", expr.LOOKUP_STARTSWITH, path),
		expr.Expr("Depth", expr.LOOKUP_EXACT, depth+1),
	)
}

func (qs *PageQuerySet) Siblings(path string, depth int64, inclusive ...bool) *PageQuerySet {
	var incl = variableBool(inclusive...)
	var parentPath, err = ancestorPath(path, 1)
	if err != nil {
		panic(errors.Wrapf(err, "failed to get parent path for %s", path))
	}

	qs = qs.Filter(
		expr.Expr("Path", expr.LOOKUP_STARTSWITH, parentPath),
		expr.Expr("Depth", expr.LOOKUP_EXACT, depth),
	)

	if !incl {
		qs = qs.Filter(
			expr.Expr("Path", expr.LOOKUP_EXACT, path).Not(true),
		)
	}

	return qs
}

func (qs *PageQuerySet) AncestorOf(node *PageNode, inclusive ...bool) *PageQuerySet {
	return qs.Ancestors(node.Path, node.Depth, inclusive...)
}

func (qs *PageQuerySet) DescendantOf(node *PageNode, inclusive ...bool) *PageQuerySet {
	return qs.Descendants(node.Path, node.Depth, inclusive...)
}

func (qs *PageQuerySet) ChildrenOf(node *PageNode) *PageQuerySet {
	return qs.Children(node.Path, node.Depth)
}

func (qs *PageQuerySet) SiblingsOf(node *PageNode, inclusive ...bool) *PageQuerySet {
	return qs.Siblings(node.Path, node.Depth, inclusive...)
}

type searchSet struct {
	querySet    *queries.QuerySet[Page]
	contentType *contenttypes.BaseContentType[Page]
}

func (qs *PageQuerySet) BuildSearchQuery(_ []search.BuiltSearchField, query string) (search.Searchable, error) {
	var defs = ListDefinitions()
	var searchableDefs = make([]*PageDefinition, 0, len(defs))
	for _, def := range defs {
		if _, ok := def.Object().(search.SearchableModel); ok {
			searchableDefs = append(searchableDefs, def)
		}
	}

	//	relevanceField, err := qs.Base().WalkField("_relevance")
	//	if err != nil {
	//		return nil, err
	//	}
	//
	//	var relevanceExpr *expr.CaseExpression
	//	if relevanceField.Annotation != nil {
	//		var ex = relevanceField.Annotation.(interface{ Expression() expr.Expression }).Expression()
	//		relevanceExpr = ex.(*expr.CaseExpression)
	//	}

	var searchSets = make([]*searchSet, 0, len(searchableDefs))
	for _, def := range searchableDefs {
		var (
			modelObj      = def.Object().(search.SearchableModel)
			meta          = attrs.GetModelMeta(modelObj)
			cType         = contenttypes.NewContentType(modelObj.(Page))
			defs          = meta.Definitions()
			primary       = defs.Primary()
			_, isPageNode = def.Object().(*PageNode)
		)

		if isPageNode {
			// we already are searching PageNodes, no need to add it again
			continue
		}

		var nestedQS, err = search.Search(modelObj, query, queries.GetQuerySet(modelObj.(Page)))
		if err != nil {
			return nil, err
		}

		fr, err := nestedQS.WalkField("_relevance")
		if err != nil {
			return nil, err
		}

		if fr.Annotation != nil {

			nestedQS = nestedQS.
				Select("_relevance").
				Filter(expr.OuterRef("PageID"), expr.Field(primary.Name())).
				Filter(expr.OuterRef("ContentType"), cType.TypeName())

			qs = qs.Annotate("_nested_relevance", queries.Subquery(nestedQS.Limit(1)))
			// qs = qs.Annotate("_nested_relevance", expr.Logical("__nested_relevance").ADD(relevanceExpr))
			// qs = qs.Annotate("_nested_relevance", queries.Subquery(nestedQS.Select("_relevance").Limit(1)))
		}

		//if fr.Annotation != nil {
		//	nestedQS = nestedQS.Select("_relevance").Filter(
		//		expr.Q(primary.Name(), expr.OuterRef("PageID")),
		//		expr.Logical(expr.Value(cType.TypeName())).EQ(expr.OuterRef("ContentType")),
		//	)
		//	// qs = qs.Annotate("_nested_relevance", queries.Subquery(nestedQS.Limit(1)))
		//	// qs = qs.Annotate("_nested_relevance", expr.Logical("__nested_relevance").ADD(relevanceExpr))
		//	// qs = qs.Annotate("_nested_relevance", queries.Subquery(nestedQS.Select("_relevance").Limit(1)))
		//	qs.BaseQuerySet = qs.Base().Union(queries.ChangeObjectsType[*PageNode, attrs.Definer](
		//		c.Clone().Annotate("_relevance", queries.Subquery(nestedQS.Limit(1))).Base().
		//			Filter(expr.And[any](
		//				expr.Q("PK__in", nestedQS.Select(primary.Name())),
		//				expr.Q("ContentType", cType.TypeName()),
		//			)),
		//	))
		//}

		searchSets = append(searchSets, &searchSet{
			contentType: cType,
			querySet:    nestedQS.Select(primary.Name()),
		})
	}

	var orExprs = make([]expr.Expression, 0, len(searchSets)) //+1)
	for _, set := range searchSets {
		orExprs = append(orExprs, expr.And(
			expr.Q("PageID__in", set.querySet),
			expr.Q("ContentType", set.contentType),
		))
	}

	//orExprs = append(orExprs, expr.Q(
	//	"PK__in", qs.Base().Select("PK"),
	//))

	if len(orExprs) > 0 {
		qs = qs.Filter(queries.PushFilterOR(
			expr.Or(orExprs...),
		))
	}

	return qs.OrderBy("-_relevance", "-_nested_relevance"), nil
}

func (qs *PageQuerySet) Search(query string) *PageQuerySet {
	if strings.TrimSpace(query) == "" {
		return qs
	}

	var err error
	qs, err = search.Search(&PageNode{}, query, qs)
	if err != nil {
		assert.Fail(
			"PageQuerySet.Search: failed to perform search: %v", err,
		)
		return nil
	}

	return qs
}

func (qs *PageQuerySet) GetChildNodes(node *PageNode, statusFlags StatusFlag, offset int32, limit int32) ([]*PageNode, error) {

	return qs.Children(node.Path, node.Depth).
		Filter("StatusFlags__bitand", statusFlags).
		Limit(int(limit)).
		Offset(int(offset)).
		OrderBy("Path").
		AllNodes()
}

func (qs *PageQuerySet) GetDescendants(path string, depth int64, statusFlags StatusFlag, offset int32, limit int32) ([]*PageNode, error) {
	return qs.Descendants(path, depth).
		Filter("StatusFlags__bitand", statusFlags).
		Limit(int(limit)).
		Offset(int(offset)).
		OrderBy("Path").
		AllNodes()
}

// AncestorNodes returns the ancestor nodes of the given node.
//
// The path is a PageNode.Path, the depth is the depth of the page.
func (qs *PageQuerySet) GetAncestors(p string, depth int64) ([]*PageNode, error) {
	return qs.Ancestors(p, depth).AllNodes()
}

// AddRoots creates new root nodes.
//
// The following conditions **must** be met for each node:
// - The node path must be empty.
// - The node title must not be empty.
// - The node title must not be empty, if not provided the page's slug (and thus URLPath) will be based on the page's title.
func (qs *PageQuerySet) AddRoots(nodes ...*PageNode) error {
	if len(nodes) == 0 {
		return fmt.Errorf("no nodes provided")
	}

	transaction, err := qs.GetOrCreateTransaction()
	if err != nil {
		return errors.Wrap(err, "failed to start transaction")
	}
	defer transaction.Rollback(qs.Context())

	previousRootNodeCount, err := qs.Filter("Depth", 0).Count()
	if err != nil {
		return err
	}

	for _, node := range nodes {
		if node.Path != "" {
			return fmt.Errorf("node path must be empty")
		}

		if node.ContentType != "" {
			var definition = DefinitionForType(node.ContentType)
			if definition != nil && definition.DisallowRoot {
				return fmt.Errorf(
					"content type %q does not allow to be a root node",
					node.ContentType,
				)
			}
		}

		node.Path = buildPathPart(previousRootNodeCount)
		if node.Title == "" {
			return fmt.Errorf("node title must not be empty")
		}

		node.SetUrlPath(nil)
		node.Depth = 0

		// this is kinda inefficient, but we really DO want to call the [models.ContextSaver.Save] method
		// to ensure that the page object is saved correctly, might there be some specific logic in said method.
		if err = qs.saveSpecific(node, true); err != nil {
			return errors.Wrap(err, "failed to save specific instance")
		}

		// Increase the count for each addition,
		// this will ensure that the next root node will have a unique path.
		previousRootNodeCount++
	}

	nodes, err = qs.ExplicitSave().Base().BulkCreate(nodes)
	if err != nil {
		return err
	}

	if err = transaction.Commit(qs.Context()); err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	return SignalRootCreated.Send(&PageNodeSignal{
		BaseSignal: BaseSignal{
			Ctx: qs.Context(),
		},
		Nodes: nodes,
	})
}

// AddRoot creates a new root node.
//
// For more information, see [PageQuerySet.AddRoots].
func (qs *PageQuerySet) AddRoot(node *PageNode) error {
	return qs.AddRoots(node)
}

// CreateChildNode creates a new child node.
//
// The parent node path must not be empty.
//
// The child node path must be empty.
//
// The child node title must not be empty, if not provided the page's slug (and thus URLPath) will be based on the page's title.
//
// The child node path is set to a new path part based on the number of children of the parent node.
func (qs *PageQuerySet) AddChildren(parent *PageNode, children ...*PageNode) error {

	if len(children) == 0 {
		return fmt.Errorf("no children provided")
	}

	var transaction, err = qs.GetOrCreateTransaction()
	if err != nil {
		return errors.Wrap(err, "failed to start transaction")
	}
	defer transaction.Rollback(qs.Context())

	if parent.Path == "" {
		return fmt.Errorf("parent path must not be empty")
	}

	var parentDefinition *PageDefinition
	if len(parent.ContentType) > 0 {
		parentDefinition = DefinitionForType(parent.ContentType)
	}

	for _, child := range children {

		if child.Path != "" {
			return fmt.Errorf("child path must be empty")
		}

		child.Title = strings.TrimSpace(child.Title)
		if child.Title == "" {
			return fmt.Errorf("child title must not be empty")
		}

		child._parent = parent
		child.SetUrlPath(parent)
		child.Path = parent.Path + buildPathPart(parent.Numchild)
		child.Depth = parent.Depth + 1

		// this is kinda inefficient, but we really DO want to call the [models.ContextSaver.Save] method
		// to ensure that the page object is saved correctly, might there be some specific logic in said method.
		if err := qs.saveSpecific(child, true); err != nil {
			return errors.Wrap(err, "failed to save specific instance")
		}

		if parentDefinition != nil && child.ContentType != "" && !parentDefinition.IsValidChildType(child.ContentType) {
			return fmt.Errorf(
				"child content type %q is not allowed under parent content type %q",
				child.ContentType, parent.ContentType,
			)
		}

		if child.ContentType != "" {
			var childDefinition = DefinitionForType(child.ContentType)
			if childDefinition != nil && parent.ContentType != "" && !childDefinition.IsValidParentType(parent.ContentType) {
				return fmt.Errorf(
					"child content type %q is not allowed under parent content type %q",
					child.ContentType, parent.ContentType,
				)
			}
		}
	}

	children, err = qs.ExplicitSave().Base().BulkCreate(children)
	if err != nil {
		return errors.Wrap(err, "failed to create child nodes")
	}

	parent.Numchild += int64(len(children))
	updated, err := qs.
		Base().
		ExplicitSave().
		Select("Numchild").
		Filter("PK", parent.PK).
		Update(parent)
	if err != nil {
		return err
	}

	if queries.IsCommitContext(qs.Context()) && updated == 0 {
		return fmt.Errorf("failed to update parent node with PK %d", parent.PK)
	}

	if err = transaction.Commit(qs.Context()); err != nil {
		return err
	}

	return SignalChildCreated.Send(&PageNodeSignal{
		BaseSignal: BaseSignal{
			Ctx: qs.Context(),
		},
		Nodes:  children,
		PageID: parent.PageID,
	})
}

// UpdateNode updates a node.
//
// This function will update the node's url path if the slug has changed.
//
// In that case, it will also update the url paths of all descendants.
func (qs *PageQuerySet) UpdateNode(node *PageNode) error {

	if node.Path == "" {
		return fmt.Errorf("node path must not be empty")
	}

	if node.PK == 0 {
		return fmt.Errorf("node id must not be zero")
	}

	node.Title = strings.TrimSpace(node.Title)
	if node.Title == "" {
		return fmt.Errorf("node title must not be empty")
	}

	transaction, err := qs.GetOrCreateTransaction()
	if err != nil {
		return errors.Wrap(err, "failed to start transaction")
	}
	defer transaction.Rollback(qs.Context())

	if err = qs.saveSpecific(node, false); err != nil {
		return errors.Wrap(err, "failed to save specific instance")
	}

	oldRecord, err := qs.GetNodeByID(node.PK)
	if err != nil {
		return errors.Wrapf(err, "failed to retrieve old record with PK %d", node.PK)
	}

	if oldRecord.Slug != node.Slug {
		var parent *PageNode

		if node.Depth > 0 {
			var parentNode, err = qs.ParentNode(node.Path, int(node.Depth))
			if err != nil {
				return errors.Wrapf(err, "failed to get parent node for node with path %s", node.Path)
			}
			parent = parentNode
		}

		newPath, oldPath := node.SetUrlPath(parent)
		err = qs.updateDescendantPaths(oldPath, newPath, node.Path, node.PK)
		if err != nil {
			return errors.Wrapf(err,
				"failed to update descendant paths for node with path %s and PK %d",
				node.Path, node.PK,
			)
		}
	}

	err = qs.updateNode(
		node,
	)
	if err != nil {
		return errors.Wrap(err, "failed to update node")
	}

	if err = transaction.Commit(qs.Context()); err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	return SignalNodeUpdated.Send(&PageNodeSignal{
		BaseSignal: BaseSignal{
			Ctx: qs.Context(),
		},
		Node:   node,
		PageID: node.PageID,
	})
}

func (qs *PageQuerySet) Create(*PageNode) (*PageNode, error) {
	panic("Create is not implemented for PageQuerySet, use AddRoots or AddChildren instead or call qs.Base() for advanced usage")
}

func (qs *PageQuerySet) Update(*PageNode, ...any) (int64, error) {
	panic("Update is not implemented for PageQuerySet, use UpdateNode instead or call qs.Base() for advanced usage")
}

func (qs *PageQuerySet) BulkCreate([]*PageNode) ([]*PageNode, error) {
	panic("BulkCreate is not implemented for PageQuerySet, use AddRoots or AddChildren instead or call qs.Base() for advanced usage")
}

func (qs *PageQuerySet) BatchCreate([]*PageNode) ([]*PageNode, error) {
	panic("BatchCreate is not implemented for PageQuerySet, use AddRoots or AddChildren instead or call qs.Base() for advanced usage")
}

func (qs *PageQuerySet) BulkUpdate(...any) (int64, error) {
	panic("BulkUpdate is not implemented for PageQuerySet, use UpdateNode instead or call qs.Base() for advanced usage")
}

func (qs *PageQuerySet) BatchUpdate(...any) (int64, error) {
	panic("BatchUpdate is not implemented for PageQuerySet, use UpdateNode instead or call qs.Base() for advanced usage")
}

func (qs *PageQuerySet) Delete(nodes ...*PageNode) (int64, error) {

	if !queries.IsCommitContext(qs.Context()) {
		return 0, nil
	}

	if qs.Base().HasWhereClause() {

		if len(nodes) > 0 {
			return 0, errors.ValueError.Wrapf(
				"QuerySet.Delete: cannot delete nodes when the QuerySet has a WHERE clause",
			)
		}

		return qs.Base().Delete(nodes...)
	}

	var transaction, err = qs.GetOrCreateTransaction()
	if err != nil {
		return 0, errors.Wrap(err, "failed to start transaction")
	}
	defer transaction.Rollback(qs.Context())

	var allNodesExpr = make([]expr.ClauseExpression, 0, len(nodes))
	var parentNodePaths = make([]string, 0, len(nodes))
	var seenParents = make(map[string]struct{}, len(nodes))
	for _, node := range nodes {
		allNodesExpr = append(allNodesExpr, expr.And(
			expr.Expr("Path", expr.LOOKUP_STARTSWITH, node.Path),
			expr.Expr("Depth", expr.LOOKUP_GTE, node.Depth),
		))

		if node.Depth > 0 {
			var parentPath, err = ancestorPath(
				node.Path, 1,
			)
			if err != nil {
				return 0, errors.Wrapf(
					err, "failed to get parent path for node with path %s", node.Path,
				)
			}

			if _, exists := seenParents[parentPath]; exists {
				continue // Skip already seen parent paths
			}

			seenParents[parentPath] = struct{}{}
			parentNodePaths = append(
				parentNodePaths,
				parentPath,
			)
		}
	}

	nodeCount, nodeIter, err := qs.Base().Filter(allNodesExpr).IterAll()
	if err != nil {
		return 0, errors.Wrap(err, "failed to query nodes")
	}

	var ids = make([]int64, 0, nodeCount)
	var seenIds = make(map[int64]struct{}, nodeCount)
	var cTypes = orderedmap.NewOrderedMap[string, []int64]()
	for row, err := range nodeIter {
		if err != nil {
			return 0, errors.Wrap(err, "failed to iterate over nodes")
		}

		if _, ok := seenIds[row.Object.PK]; ok {
			continue // Skip already seen nodes
		}

		seenIds[row.Object.PK] = struct{}{}
		ids = append(ids, row.Object.PK)

		if row.Object.ContentType != "" {
			var idList, ok = cTypes.Get(row.Object.ContentType)
			if !ok {
				idList = make([]int64, 0, 1)
			}
			idList = append(idList, row.Object.PageID)
			cTypes.Set(row.Object.ContentType, idList)
		}

		var err = SignalNodeBeforeDelete.Send(&PageNodeSignal{
			BaseSignal: BaseSignal{
				Ctx: qs.Context(),
			},
			Node:   row.Object,
			PageID: row.Object.PageID,
		})

		if err != nil {
			return 0, fmt.Errorf(
				"error in before delete signal for node %d: %w", row.Object.PK, err,
			)
		}
	}

	if len(ids) == 0 {
		return 0, errors.New(errors.CodeNoRows, "no nodes to delete")
	}

	for head := cTypes.Front(); head != nil; head = head.Next() {
		var definition = DefinitionForType(head.Key)
		if definition == nil {
			return 0, errors.New(errors.CodeNoRows, fmt.Sprintf(
				"no content type definition found for %s",
				head.Key,
			))
		}

		var model = definition.Object().(Page)
		var defs = model.FieldDefs()
		var primaryField = defs.Primary()

		var (
			filterName  string
			filterValue any = head.Value
		)
		if len(head.Value) == 1 {
			filterName = primaryField.Name()
			filterValue = head.Value[0]
		} else {
			filterName = fmt.Sprintf("%s__in", primaryField.Name())
			filterValue = head.Value
		}

		var deleted, err = queries.GetQuerySetWithContext(qs.Context(), model).
			Filter(filterName, filterValue).
			Delete()
		if err != nil {
			return 0, errors.Wrapf(err, "failed to delete %s nodes", head.Key)
		}

		if deleted == 0 {
			return 0, errors.NoChanges.Wrapf(
				"failed to delete %s nodes with ids %v", head.Key, head.Value,
			)
		}
	}

	// Delete the nodes from the database
	deleted, err := qs.Base().
		Filter(allNodesExpr).
		Delete()
	if err != nil {
		return 0, errors.Wrap(err, "failed to delete nodes")
	}

	if deleted == 0 {
		return 0, errors.NoChanges.Wrapf(
			"failed to delete nodes with paths %v", parentNodePaths,
		)
	}

	if len(parentNodePaths) > 0 {

		var (
			filterName  string
			filterValue any = parentNodePaths[0]
		)
		if len(parentNodePaths) == 1 {
			filterName = "Path"
			filterValue = parentNodePaths[0]
		} else {
			filterName = "Path__in"
			filterValue = parentNodePaths
		}

		// Custom query to decrement the Numchild field of parent nodes
		// decrementNumChild does not query by Path, but by PK,
		var ct, err = qs.
			Base().
			Select("Numchild").
			Filter(filterName, filterValue).
			ExplicitSave().
			BulkUpdate(
				expr.As("Numchild", expr.Logical("Numchild").SUB(1)),
			)
		if err != nil {
			return deleted, errors.Wrap(err, "failed to decrement numchild for parent nodes")
		}

		if ct == 0 {
			return deleted, errors.NoChanges.Wrapf(
				"failed to decrement numchild for parent nodes with paths %v", parentNodePaths,
			)
		}
	}

	return deleted, transaction.Commit(qs.Context())
}

// MoveNode moves a node to a new parent.
//
// The node and new parent paths must not be empty or equal.
//
// The new parent must not be a descendant of the node.
//
// This function will update the url paths of all descendants, as well as the tree paths of the node and its descendants.
func (qs *PageQuerySet) MoveNode(node *PageNode, newParent *PageNode) error {

	if node.Path == "" {
		return fmt.Errorf("node path must not be empty: %q", node.Title)
	}

	if newParent.Path == "" {
		return fmt.Errorf("new parent path must not be empty: %q", newParent.Title)
	}

	if node.Path == newParent.Path {
		return fmt.Errorf("node and new parent paths must not be the same: %q", node.Path)
	}

	if node.Depth == 0 {
		return fmt.Errorf("node %q is a root node", node.Title)
	}

	if strings.HasPrefix(newParent.Path, node.Path) {
		return fmt.Errorf("new parent is a descendant of the node: %q", newParent.Title)
	}

	var parentDefinition *PageDefinition
	if newParent.ContentType != "" {
		parentDefinition = DefinitionForType(newParent.ContentType)
		if parentDefinition == nil {
			return fmt.Errorf("no definition found for new parent content type %q", newParent.ContentType)
		}

		if node.ContentType != "" && !parentDefinition.IsValidChildType(node.ContentType) {
			return fmt.Errorf(
				"node content type %q is not allowed under new parent content type %q",
				node.ContentType, newParent.ContentType,
			)
		}
	}

	if node.ContentType != "" {
		var childDefinition = DefinitionForType(node.ContentType)
		if childDefinition == nil {
			return fmt.Errorf("no definition found for node content type %q", node.ContentType)
		}

		if newParent.ContentType != "" && !childDefinition.IsValidParentType(newParent.ContentType) {
			return fmt.Errorf(
				"node content type %q is not allowed under new parent content type %q",
				node.ContentType, newParent.ContentType,
			)
		}
	}

	var tx, err = qs.GetOrCreateTransaction()
	if err != nil {
		return errors.Wrap(err, "failed to start transaction")
	}
	defer tx.Rollback(qs.Context())

	oldParentPath, err := ancestorPath(node.Path, 1)
	if err != nil {
		return errors.Wrap(err, "failed to get old parent path")
	}

	oldParent, err := qs.GetNodeByPath(oldParentPath)
	if err != nil {
		return errors.Wrap(err, "failed to get old parent node")
	}

	nodes, err := qs.GetDescendants(node.Path, node.Depth-1, StatusFlagNone, 0, 1000)
	if err != nil && !errors.Is(err, errors.NoRows) {
		return errors.Wrap(err, "failed to get descendants")
	}

	var nextPathPart = buildPathPart(int64(newParent.Numchild))

	for _, descendant := range nodes {
		// Replace the old path part with the new path part
		// e.g. if the node path is "000100020003" and the new parent path is "00020001"
		// and the new path part is "0003" (the parent has 2 children), the new path will be "000200010003"
		descendant.Path = newParent.Path + nextPathPart + descendant.Path[(node.Depth+1)*STEP_LEN:]
		descendant.Depth = (newParent.Depth + descendant.Depth + 1) - node.Depth
	}

	updated, err := qs.
		Base().
		Select("Path", "Depth").
		ExplicitSave().
		BulkUpdate(nodes)

	if err != nil {
		return errors.Wrap(err, "failed to update descendants")
	}

	if queries.IsCommitContext(qs.Context()) && updated == 0 {
		return errors.NoChanges.Wrapf("failed to update descendants for node with path %s", node.Path)
	}

	// Update url paths of descendants
	var newPath, oldPath = node.SetUrlPath(newParent)
	node.Path = newParent.Path + nextPathPart
	node.Depth = newParent.Depth + 1

	if err = qs.updateNode(node); err != nil {
		return errors.Wrap(err, "failed to update node")
	}

	if err = qs.updateDescendantPaths(oldPath, newPath, node.Path, node.PK); err != nil {
		return errors.Wrap(err, "failed to update descendant paths")
	}

	// Update numchild of old and new parent
	newParent.Numchild++
	oldParent.Numchild--

	n, err := qs.Base().
		Select("Numchild").
		ExplicitSave().
		BulkUpdate([]*PageNode{
			newParent,
			oldParent,
		})
	if err != nil {
		return errors.Wrap(err, "failed to update parent nodes")
	}
	if queries.IsCommitContext(qs.Context()) && n == 0 {
		return errors.NoChanges.Wrapf("failed to update parent nodes with PKs %d and %d", newParent.PK, oldParent.PK)
	}

	if err = tx.Commit(qs.Context()); err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	return SignalNodeMoved.Send(&PageMovedSignal{
		BaseSignal: BaseSignal{
			Ctx: qs.Context(),
		},
		Node:      node,
		Nodes:     nodes,
		OldParent: oldParent,
		NewParent: newParent,
	})
}

// PublishNode will set the published flag on the node
// and update it accordingly in the database.
func (qs *PageQuerySet) PublishNode(node *PageNode) error {
	if node.Path == "" {
		return fmt.Errorf("node path must not be empty")
	}

	if node.StatusFlags.Is(StatusFlagPublished) {
		return nil
	}

	node.StatusFlags |= StatusFlagPublished
	return qs.updateNodeStatusFlags(int64(StatusFlagPublished), node.PK)
}

// UnpublishNode will unset the published flag on the node
// and update it accordingly in the database.
//
// If unpublishChildren is true, it will also unpublish all descendants.
func (qs *PageQuerySet) UnpublishNode(node *PageNode, unpublishChildren bool) error {
	if node.Path == "" {
		return fmt.Errorf("node path must not be empty")
	}

	if !queries.IsCommitContext(qs.Context()) {
		return nil
	}

	var transaction, err = qs.GetOrCreateTransaction()
	if err != nil {
		return errors.Wrap(err, "failed to start transaction")
	}
	defer transaction.Rollback(qs.Context())

	if !node.StatusFlags.Is(StatusFlagPublished) {
		return nil
	}

	var xp expr.ClauseExpression = expr.Q("PK", node.PK)
	if unpublishChildren {
		xp = expr.Or[any](
			xp,
			expr.And(
				expr.Q("StatusFlags__bitand", int64(StatusFlagPublished)),
				expr.Q("Path__startswith", node.Path),
				expr.Q("Depth__gt", node.Depth),
			),
		)
	}

	updated, err := qs.
		Base().
		ExplicitSave().
		Select("StatusFlags").
		Filter(xp).
		BulkUpdate(
			expr.As(
				"StatusFlags",
				expr.Expr("StatusFlags", expr.LOOKUP_BITAND, ^int64(StatusFlagPublished)),
			),
		)
	if err != nil {
		return errors.Wrap(err, "failed to update node status flags")
	}
	if updated == 0 {
		return errors.NoChanges.Wrapf("failed to unpublish node with PK %d", node.PK)
	}

	return transaction.Commit(qs.Context())
}

// ParentNode returns the parent node of the given node.
func (qs *PageQuerySet) ParentNode(path string, depth int) (v *PageNode, err error) {
	if depth == 0 {
		return v, ErrPageIsRoot
	}
	var parentPath string
	parentPath, err = ancestorPath(
		path, 1,
	)
	if err != nil {
		return v, err
	}
	return qs.GetNodeByPath(
		parentPath,
	)
}

func (qs *PageQuerySet) AllNodes() ([]*PageNode, error) {
	var rows, err = qs.All()
	if err != nil {
		return nil, fmt.Errorf("failed to query nodes: %w", err)
	}
	return rowsToNodes(rows), nil
}

func (qs *PageQuerySet) GetNodeByID(id int64) (*PageNode, error) {
	var row, err = qs.
		Filter("PK", id).
		Get()
	if err != nil {
		return nil, err
	}
	return row.Object, nil
}

func (qs *PageQuerySet) GetNodeByPath(path string) (*PageNode, error) {

	var row, err = qs.
		Filter("Path", path).
		Get()
	if err != nil {
		return nil, err
	}
	return row.Object, nil
}

func (qs *PageQuerySet) GetNodeBySlug(slug string, depth int64, path string) (*PageNode, error) {

	var row, err = qs.
		Filter(
			expr.Expr("Depth", expr.LOOKUP_EXACT, depth),
			expr.Expr("Slug", expr.LOOKUP_IEXACT, slug),
			expr.Expr("Path", expr.LOOKUP_STARTSWITH, path),
		).
		Get()
	if err != nil {
		return nil, err
	}
	return row.Object, nil
}

func (qs *PageQuerySet) GetNodesByDepth(depth int64, statusFlags StatusFlag, offset int32, limit int32) ([]*PageNode, error) {

	return qs.StatusFlags(statusFlags).
		Filter(
			expr.Expr("Depth", expr.LOOKUP_EXACT, depth),
		).
		Limit(int(limit)).
		Offset(int(offset)).
		OrderBy("Path").
		AllNodes()
}

func (qs *PageQuerySet) GetNodesByIDs(id []int64) ([]*PageNode, error) {
	return qs.Filter("PK__in", id).AllNodes()
}

func rowsToNodes(rows queries.Rows[*PageNode]) []*PageNode {
	var nodes = make([]*PageNode, 0, len(rows))
	for obj := range rows.Objects() {
		nodes = append(nodes, obj)
	}
	return nodes
}

func (qs *PageQuerySet) statusFlags(statusFlags StatusFlag) expr.ClauseExpression {
	// return qs.Filter("StatusFlags__bitand", statusFlags)
	return expr.Expr("StatusFlags", expr.LOOKUP_BITAND, statusFlags)
}

func (qs *PageQuerySet) saveSpecific(node *PageNode, creating bool) error {

	if _, ok := node.PageObject.(*PageNode); ok || node.PageObject == nil {
		return nil
	}

	if creating || node.ContentType == "" {
		node.ContentType = contenttypes.
			NewContentType(node.PageObject).
			TypeName()
	}

	err := node.PageObject.Save(qs.Context())
	if err != nil {
		return errors.Wrap(err, "failed to save page object")
	}

	if !creating || node.PageID != 0 {
		return nil
	}

	var (
		srcDefs     = node.PageObject.FieldDefs()
		dstDefs     = node.FieldDefs()
		refField, _ = dstDefs.Field("PageID")
		srcVal, _   = srcDefs.Primary().Value()
	)

	return refField.Scan(srcVal)
}

func (qs *PageQuerySet) updateNodes(nodes []*PageNode, exclude ...string) error {

	var fields = []string{"PK", "Title", "Path", "Depth", "Numchild", "UrlPath", "Slug", "StatusFlags", "PageID", "ContentType", "LatestRevisionCreatedAt", "UpdatedAt"}
	fields = attrs.FieldNames(fields, exclude)

	var updated, err = qs.
		Base().
		ExplicitSave().
		Select(attrutils.InterfaceList(fields)...).
		BulkUpdate(nodes)
	if err != nil {
		return fmt.Errorf("failed to prepare nodes for update: %w", err)
	}
	if queries.IsCommitContext(qs.Context()) && updated == 0 {
		return errors.New(errors.CodeNoChanges, "no nodes were updated")
	}
	return nil
}

func (qs *PageQuerySet) updateDescendantPaths(oldUrlPath, newUrlPath, pageNodePath string, id int64) error {
	//Annotate(
	//	"ChildCount",
	//	expr.COUNT(queries.GetQuerySet(&PageNode{}).
	//		Filter("Path__startswith", pageNodePath).
	//		Filter("Depth__gt", expr.Logical(expr.LENGTH(expr.V(pageNodePath))).ADD(expr.V(1, true)))),
	//).

	if pageNodePath == "" {
		return fmt.Errorf("page node path must not be empty")
	}

	var _, err = qs.
		Base().
		Select("UrlPath").
		Filter(
			expr.Expr("Path", expr.LOOKUP_STARTSWITH, pageNodePath),
			expr.Expr("PK", expr.LOOKUP_NOT, id),
		).
		ExplicitSave().
		BulkUpdate(
			expr.As(
				// ![UrlPath] = CONCAT(?, SUBSTRING(![UrlPath], LENGTH(?) + 1))
				"UrlPath",
				expr.CONCAT(
					expr.V(newUrlPath),
					expr.SUBSTR("UrlPath", expr.Logical(expr.LENGTH(expr.V(oldUrlPath))).ADD(expr.V(1, true)), nil),
				),
			),
		)
	if err != nil && !errors.Is(err, errors.NoChanges) {
		return fmt.Errorf("failed to update descendant paths: %w", err)
	}
	//if updated == 0 {
	//	return errors.New(errors.CodeNoChanges, "no descendant paths were updated")
	//}
	return nil
}

func (qs *PageQuerySet) incrementNumChild(id ...int64) error {
	var (
		filterName  string
		filterValue any
	)
	if len(id) == 1 {
		filterName = "PK"
		filterValue = id[0]
	} else {
		filterName = "PK__in"
		filterValue = id
	}

	var ct, err = qs.
		Base().
		Select("Numchild").
		Filter(filterName, filterValue).
		ExplicitSave().
		BulkUpdate(
			expr.As("Numchild", expr.Logical("Numchild").ADD(1)),
		)
	if err != nil {
		return fmt.Errorf("failed to increment numchild: %w", err)
	}
	if queries.IsCommitContext(qs.Context()) && ct == 0 {
		return fmt.Errorf("no nodes were updated for id %d", id)
	}
	return nil
}

func (qs *PageQuerySet) decrementNumChild(id ...int64) error {
	var (
		filterName  string
		filterValue any
	)
	if len(id) == 1 {
		filterName = "PK"
		filterValue = id[0]
	} else {
		filterName = "PK__in"
		filterValue = id
	}

	var ct, err = qs.
		Base().
		Select("Numchild").
		Filter(filterName, filterValue).
		ExplicitSave().
		BulkUpdate(
			expr.As("Numchild", expr.Logical("Numchild").SUB(1)),
		)
	if err != nil {
		return fmt.Errorf("failed to decrement numchild: %w", err)
	}
	if queries.IsCommitContext(qs.Context()) && ct == 0 {
		return fmt.Errorf("no nodes were updated for id %d", id)
	}
	return nil
}

func (qs *PageQuerySet) updateNode(node *PageNode) error {

	var base = qs.Base()
	var info = base.Peek()

	if len(info.Select) == 0 {
		base = base.Select("PK", "Title", "Path", "Depth", "Numchild", "UrlPath", "Slug", "StatusFlags", "PageID", "ContentType", "PublishedAt", "LatestRevisionCreatedAt", "UpdatedAt")
	}

	updated, err := base.
		Filter("PK", node.PK).
		ExplicitSave().
		Update(node)
	if err != nil {
		return err
	}

	if queries.IsCommitContext(qs.Context()) && updated == 0 {
		// still commit the transaction as opposed to rolling it back
		// some databases might have issues reporting back the amount of updated rows
		return errors.NoChanges
	}

	return nil
}

func (qs *PageQuerySet) updateNodeStatusFlags(statusFlags int64, iD int64) error {

	transaction, err := qs.GetOrCreateTransaction()
	if err != nil {
		return errors.Wrap(err, "failed to start transaction")
	}
	defer transaction.Rollback(qs.Context())

	updated, err := qs.
		Base().
		Select("StatusFlags").
		Filter("PK", iD).
		ExplicitSave().
		Update(
			&PageNode{
				StatusFlags: StatusFlag(statusFlags),
			},
			expr.F("![UpdatedAt] = CURRENT_TIMESTAMP"),
		)
	if err != nil {
		return err
	}

	if queries.IsCommitContext(qs.Context()) && updated == 0 {
		// still commit the transaction as opposed to rolling it back
		// some databases might have issues reporting back the amount of updated rows
		return errors.Join(
			errors.NoChanges,
			transaction.Commit(qs.Context()),
		)
	}

	return transaction.Commit(qs.Context())
}
