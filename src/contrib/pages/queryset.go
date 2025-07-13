package pages

import (
	"fmt"
	"strings"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/attrs"
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
	var pageQuerySet = &PageQuerySet{}
	pageQuerySet.WrappedQuerySet = queries.WrapQuerySet(
		queries.GetQuerySet(&PageNode{}),
		pageQuerySet,
	)
	return pageQuerySet
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
		exp = expr.Or(
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

// CreateRootNode creates a new root node.
//
// The node path must be empty.
//
// The node title must not be empty.
//
// The child node title must not be empty, if not provided the page's slug (and thus URLPath) will be based on the page's title.
//
// The node path is set to a new path part based on the number of root nodes.
func (qs *PageQuerySet) AddRoot(node *PageNode) error {

	if node.Path != "" {
		return fmt.Errorf("node path must be empty")
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

	node.Path = buildPathPart(previousRootNodeCount)
	if node.Title == "" {
		return fmt.Errorf("node title must not be empty")
	}

	node.SetUrlPath(nil)
	node.Depth = 0

	if err = qs.saveSpecific(node, true); err != nil {
		return errors.Wrap(err, "failed to save specific instance")
	}

	node.PK, err = qs.insertNode(node)
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
		Node: node,
	})
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
func (qs *PageQuerySet) AddChild(parent, child *PageNode) error {

	var transaction, err = qs.GetOrCreateTransaction()
	if err != nil {
		return errors.Wrap(err, "failed to start transaction")
	}
	defer transaction.Rollback(qs.Context())

	if parent.Path == "" {
		return fmt.Errorf("parent path must not be empty")
	}

	if child.Path != "" {
		return fmt.Errorf("child path must be empty")
	}

	child.Title = strings.TrimSpace(child.Title)
	if child.Title == "" {
		return fmt.Errorf("child title must not be empty")
	}

	child.SetUrlPath(parent)
	child.Path = parent.Path + buildPathPart(parent.Numchild)
	child.Depth = parent.Depth + 1

	if err := qs.saveSpecific(child, true); err != nil {
		return errors.Wrap(err, "failed to save specific instance")
	}

	child.PK, err = qs.insertNode(child)
	if err != nil {
		return err
	}

	parent.Numchild++
	updated, err := qs.
		ExplicitSave().
		Select("Numchild").
		Filter("PK", parent.PK).
		Update(parent)
	if err != nil {
		return err
	}

	if updated == 0 {
		return fmt.Errorf("failed to update parent node with PK %d", parent.PK)
	}

	if err = transaction.Commit(qs.Context()); err != nil {
		return err
	}

	return SignalChildCreated.Send(&PageNodeSignal{
		BaseSignal: BaseSignal{
			Ctx: qs.Context(),
		},
		Node:   child,
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

		node.SetUrlPath(parent)
		err = qs.updateDescendantPaths(oldRecord.UrlPath, node.UrlPath, node.Path, node.PK)
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

// DeleteRootNode deletes a root node.
func (qs *PageQuerySet) DeleteRootNode(node *PageNode) error {

	if node.Depth != 0 {
		return fmt.Errorf("node is not a root node")
	}

	if node.Path == "" {
		return fmt.Errorf("node path must not be empty")
	}

	transaction, err := qs.GetOrCreateTransaction()
	if err != nil {
		return errors.Wrap(err, "failed to start transaction")
	}
	defer transaction.Rollback(qs.Context())

	descendants, err := qs.GetDescendants(
		node.Path, node.Depth+1, StatusFlagNone, 0, 1000,
	)
	if err != nil {
		return errors.Wrap(err, "failed to get descendants")
	}

	err = qs.deleteNodes(append(descendants, node))
	if err != nil {
		return errors.Wrap(err, "failed to delete nodes")
	}

	return transaction.Commit(qs.Context())
}

// DeleteNode deletes a page node.
func (qs *PageQuerySet) DeleteNode(node *PageNode) error { //, newParent *PageNode) error {
	if node.Depth == 0 {
		return qs.DeleteRootNode(node)
	}

	var tx, err = qs.GetOrCreateTransaction()
	if err != nil {
		return errors.Wrap(err, "failed to start transaction")
	}
	defer tx.Rollback(qs.Context())

	parentPath, err := ancestorPath(
		node.Path, 1,
	)
	if err != nil {
		return errors.Wrapf(err, "failed to get parent path for node with path %s", node.Path)
	}

	parent, err := qs.GetNodeByPath(
		parentPath,
	)
	if err != nil {
		return errors.Wrapf(err, "failed to get parent node for node with path %s", node.Path)
	}

	var descendants []*PageNode
	descendants, err = qs.GetDescendants(
		node.Path, node.Depth-1, StatusFlagNone, 0, 1000,
	)
	if err != nil {
		return errors.Wrap(err, "failed to get descendants")
	}

	err = qs.deleteNodes(descendants)
	if err != nil {
		return errors.Wrap(err, "failed to delete descendants")
	}

	err = qs.decrementNumChild(parent.PK)
	if err != nil {
		return errors.Wrap(err, "failed to decrement parent numchild")
	}

	return tx.Commit(qs.Context())
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
		return fmt.Errorf("node path must not be empty")
	}

	if newParent.Path == "" {
		return fmt.Errorf("new parent path must not be empty")
	}

	if node.Path == newParent.Path {
		return fmt.Errorf("node and new parent paths must not be the same")
	}

	if node.Depth == 0 {
		return fmt.Errorf("node is a root node")
	}

	if strings.HasPrefix(newParent.Path, node.Path) {
		return fmt.Errorf("new parent is a descendant of the node")
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
	if err != nil {
		return errors.Wrap(err, "failed to get descendants")
	}

	for _, descendant := range nodes {
		descendant := descendant
		descendant.Path = newParent.Path + descendant.Path[node.Depth*STEP_LEN:]
		descendant.Depth = (newParent.Depth + descendant.Depth + 1) - node.Depth
	}

	updated, err := qs.
		Select("Path", "Depth").
		ExplicitSave().
		BulkUpdate(nodes)

	if err != nil {
		return errors.Wrap(err, "failed to update descendants")
	}

	if updated == 0 {
		return errors.NoChanges.Wrapf("failed to update descendants for node with path %s", node.Path)
	}

	// Update url paths of descendants
	var newPath, oldPath = node.SetUrlPath(newParent)
	node.Path = newParent.Path + buildPathPart(int64(
		newParent.Numchild,
	))
	node.Depth = newParent.Depth + 1

	if err = qs.updateNode(node); err != nil {
		return errors.Wrap(err, "failed to update node")
	}

	if err = qs.updateDescendantPaths(oldPath, newPath, node.Path, node.PK); err != nil {
		return errors.Wrap(err, "failed to update descendant paths")
	}

	err = qs.incrementNumChild(newParent.PK)
	if err != nil {
		return errors.Wrap(err, "failed to increment new parent numchild")
	}
	newParent.Numchild++

	err = qs.decrementNumChild(oldParent.PK)
	if err != nil {
		return errors.Wrap(err, "failed to decrement old parent numchild")
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
		xp = expr.Or(
			xp,
			expr.And(
				expr.Q("StatusFlags__bitand", int64(StatusFlagPublished)),
				expr.Q("Path__startswith", node.Path),
				expr.Q("Depth__gt", node.Depth),
			),
		)
	}

	updated, err := qs.
		ExplicitSave().
		Select("StatusFlags").
		Filter(xp).
		Update(
			&PageNode{},
			expr.Expr("StatusFlags", expr.LOOKUP_BITAND, ^int64(StatusFlagPublished)),
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

func (qs *PageQuerySet) updateNodes(nodes []*PageNode) error {

	var updated, err = qs.
		ExplicitSave().
		Select("PK", "Title", "Path", "Depth", "Numchild", "UrlPath", "Slug", "StatusFlags", "PageID", "ContentType", "LatestRevisionID", "UpdatedAt").
		BulkUpdate(nodes)
	if err != nil {
		return fmt.Errorf("failed to prepare nodes for update: %w", err)
	}
	if updated == 0 {
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

	var _, err = qs.
		Select("UrlPath").
		Filter(
			expr.Expr("Path", expr.LOOKUP_STARTSWITH, pageNodePath),
			expr.Expr("PK", expr.LOOKUP_NOT, id),
		).
		ExplicitSave().
		Update(
			&PageNode{},
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

func (qs *PageQuerySet) incrementNumChild(id int64) error {
	var ct, err = qs.
		Select("Numchild").
		Filter("PK", id).
		ExplicitSave().
		Update(
			&PageNode{},
			expr.As("Numchild", expr.Logical("Numchild").ADD(1)),
		)
	if err != nil {
		return fmt.Errorf("failed to increment numchild: %w", err)
	}
	if ct == 0 {
		return fmt.Errorf("no nodes were updated for id %d", id)
	}
	return nil
}

func (qs *PageQuerySet) decrementNumChild(id int64) error {
	var ct, err = qs.
		Select("Numchild").
		Filter("PK", id).
		ExplicitSave().
		Update(
			&PageNode{},
			expr.As("Numchild", expr.Logical("Numchild").SUB(1)),
		)
	if err != nil {
		return fmt.Errorf("failed to decrement numchild: %w", err)
	}
	if ct == 0 {
		return fmt.Errorf("no nodes were updated for id %d", id)
	}
	return nil
}

func (qs *PageQuerySet) deleteNodes(nodes []*PageNode) error {

	if !qs.Compiler().InTransaction() && queries.QUERYSET_CREATE_IMPLICIT_TRANSACTION {
		panic("deleteNodes should be called within a transaction")
	}

	var ids = make([]int64, len(nodes))
	var cTypes = orderedmap.NewOrderedMap[string, []int64]()
	for i, node := range nodes {
		ids[i] = node.PK

		if node.ContentType != "" {
			var idList, ok = cTypes.Get(node.ContentType)
			if !ok {
				idList = make([]int64, 0, 1)
			}
			idList = append(idList, node.PK)
			cTypes.Set(node.ContentType, idList)
		}

		var err = SignalNodeBeforeDelete.Send(&PageNodeSignal{
			BaseSignal: BaseSignal{
				Ctx: qs.Context(),
			},
			Node:   node,
			PageID: node.PageID,
		})

		if err != nil {
			return fmt.Errorf(
				"error in before delete signal for node %d: %w", node.PK, err,
			)
		}
	}

	if len(ids) == 1 {
		qs = qs.Filter("PK", ids[0])
	} else if len(ids) > 1 {
		qs = qs.Filter("PK__in", ids)
	}

	var deleted, err = qs.Delete()
	if err != nil {
		return err
	}

	for head := cTypes.Front(); head != nil; head = head.Next() {
		var definition = DefinitionForType(head.Key)
		if definition == nil {
			return errors.New(errors.CodeNoRows, fmt.Sprintf(
				"no content type definition found for %s",
				head.Key,
			))
		}

		var model = definition.Object().(Page)
		var defs = model.FieldDefs()
		var primaryField = defs.Primary()
		var deleted, err = queries.GetQuerySetWithContext(qs.Context(), model).
			Filter(fmt.Sprintf("%s__in", primaryField.Name()), head.Value).
			Delete()
		if err != nil {
			return errors.Wrapf(err, "failed to delete %s nodes", head.Key)
		}

		if deleted == 0 {
			return errors.New(errors.CodeNoRows, fmt.Sprintf(
				"no %s nodes were deleted",
				head.Key,
			))
		}
	}

	if deleted == 0 {
		return errors.NoRows
	}

	return err
}

func (qs *PageQuerySet) insertNode(node *PageNode) (int64, error) {
	var err error

	node, err = qs.
		ExplicitSave().
		Create(node)
	if err != nil {
		return 0, err
	}
	return node.PK, nil
}

func (qs *PageQuerySet) updateNode(node *PageNode) error {

	updated, err := qs.
		Select("PK", "Title", "Path", "Depth", "Numchild", "UrlPath", "Slug", "StatusFlags", "PageID", "ContentType", "LatestRevisionID", "UpdatedAt").
		Filter("PK", node.PK).
		ExplicitSave().
		Update(node)
	if err != nil {
		return err
	}

	if updated == 0 {
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

	if updated == 0 {
		// still commit the transaction as opposed to rolling it back
		// some databases might have issues reporting back the amount of updated rows
		return errors.Join(
			errors.NoChanges,
			transaction.Commit(qs.Context()),
		)
	}

	return transaction.Commit(qs.Context())
}
