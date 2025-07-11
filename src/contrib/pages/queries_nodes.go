package pages

import (
	"fmt"
	"strings"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
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

func (qs *PageQuerySet) StatusFlags(statusFlags StatusFlag) *PageQuerySet {
	return qs.Filter("StatusFlags__bitand", statusFlags)
}

func (qs *PageQuerySet) Ancestors(parentPath string, depth int64) *PageQuerySet {
	depth++

	var paths = make([]string, depth)
	for i := 1; i < int(depth); i++ {
		var path, err = ancestorPath(
			parentPath, int64(i),
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

func (qs *PageQuerySet) Descendants(path string, depth int64) *PageQuerySet {
	return qs.Filter(
		expr.Expr("Path", expr.LOOKUP_STARTSWITH, path),
		expr.Expr("Depth", expr.LOOKUP_GT, depth),
	)
}

func (qs *PageQuerySet) Children(path string, depth int64) *PageQuerySet {
	return qs.Filter(
		expr.Expr("Path", expr.LOOKUP_STARTSWITH, path),
		expr.Expr("Depth", expr.LOOKUP_EXACT, depth+1),
	)
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
	qs = qs.Reset()

	if node.Path != "" {
		return fmt.Errorf("node path must be empty")
	}

	transaction, err := qs.GetOrCreateTransaction()
	if err != nil {
		return errors.Wrap(err, "failed to start transaction")
	}
	defer transaction.Rollback(qs.Context())

	previousRootNodeCount, err := qs.CountRootNodes(StatusFlagNone)
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

	qs = qs.Reset()

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
	qs = qs.Reset()

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
	qs = qs.Reset()

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

	var ids = make([]int64, len(descendants))
	for i, descendant := range descendants {
		if err = SignalNodeBeforeDelete.Send(&PageNodeSignal{
			BaseSignal: BaseSignal{
				Ctx: qs.Context(),
			},
			Node:   descendant,
			PageID: node.PageID,
		}); err != nil {
			return err
		}
		ids[i] = descendant.PK
	}

	err = qs.deleteNodes(append(ids, node.PK))
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

	qs = qs.Reset()

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

	var ids = make([]int64, len(descendants))
	for i, descendant := range descendants {
		if err = SignalNodeBeforeDelete.Send(&PageNodeSignal{
			BaseSignal: BaseSignal{
				Ctx: qs.Context(),
			},
			Node:   descendant,
			PageID: node.PageID,
		}); err != nil {
			return err
		}
		ids[i] = descendant.PK
	}

	err = qs.deleteNodes(ids)
	if err != nil {
		return errors.Wrap(err, "failed to delete descendants")
	}

	prnt, err := qs.decrementNumChild(parent.PK)
	if err != nil {
		return errors.Wrap(err, "failed to decrement parent numchild")
	}
	*parent = *prnt

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
	qs = qs.Reset()

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
		Reset().
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

	prnt, err := qs.incrementNumChild(newParent.PK)
	if err != nil {
		return errors.Wrap(err, "failed to increment new parent numchild")
	}
	*newParent = *prnt

	_, err = qs.decrementNumChild(oldParent.PK)
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
