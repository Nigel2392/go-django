package pages

import (
	"context"
	"fmt"
	"strings"

	models "github.com/Nigel2392/go-django/src/contrib/pages/page_models"
	"github.com/pkg/errors"
)

// CreateRootNode creates a new root node.
//
// The node path must be empty.
//
// The node title must not be empty.
//
// The child node title must not be empty, if not provided the page's slug (and thus URLPath) will be based on the page's title.
//
// The node path is set to a new path part based on the number of root nodes.
func CreateRootNode(q models.Querier, ctx context.Context, node *models.PageNode) error {
	if node.Path != "" {
		return fmt.Errorf("node path must be empty")
	}

	previousRootNodeCount, err := q.CountRootNodes(ctx, models.StatusFlagNone)
	if err != nil {
		return err
	}

	node.Path = buildPathPart(previousRootNodeCount)
	if node.Title == "" {
		return fmt.Errorf("node title must not be empty")
	}

	node.SetUrlPath(nil)
	node.Depth = 0

	id, err := q.InsertNode(ctx, node.Title, node.Path, node.Depth, node.Numchild, node.UrlPath, node.Slug, int64(node.StatusFlags), node.PageID, node.ContentType, node.LatestRevisionID)
	if err != nil {
		return err
	}

	node.PK = id

	return SignalRootCreated.Send(&PageNodeSignal{
		BaseSignal: BaseSignal{
			Querier: q,
			Ctx:     ctx,
		},
		Node: node,
	})
}

// CountNodesByType returns the number of nodes with the given content type.
func CountNodesByType(q models.Querier, ctx context.Context, contentType string) (int64, error) {
	return q.CountNodesByTypeHash(ctx, contentType)
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
func CreateChildNode(q models.DBQuerier, ctx context.Context, parent, child *models.PageNode) error {

	var prepped, err = PrepareQuerySet(ctx, q.DB())
	if prepped != nil {
		defer prepped.Close()
	}
	if err != nil {
		return err
	}

	tx, err := prepped.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	var queries = prepped.WithTx(tx)
	defer tx.Rollback()

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
	child.PK, err = queries.InsertNode(ctx, child.Title, child.Path, child.Depth, child.Numchild, child.UrlPath, child.Slug, int64(child.StatusFlags), child.PageID, child.ContentType, child.LatestRevisionID)
	if err != nil {
		return err
	}

	parent.Numchild++
	*parent, err = queries.IncrementNumChild(ctx, parent.PK)
	if err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return SignalChildCreated.Send(&PageNodeSignal{
		BaseSignal: BaseSignal{
			Querier: q,
			Ctx:     ctx,
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
func UpdateNode(q models.Querier, ctx context.Context, node *models.PageNode) error {
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

	var oldRecord, err = q.GetNodeByID(ctx, node.PK)
	if err != nil {
		return errors.Wrapf(err, "failed to retrieve old record with PK %d", node.PK)
	}

	if oldRecord.Slug != node.Slug {
		var parent *models.PageNode

		if node.Depth > 0 {
			var parentNode, err = ParentNode(q, ctx, node.Path, int(node.Depth))
			if err != nil {
				return errors.Wrapf(err, "failed to get parent node for node with path %s", node.Path)
			}
			parent = &parentNode
		}

		node.SetUrlPath(parent)
		err = q.UpdateDescendantPaths(ctx, oldRecord.UrlPath, node.UrlPath, node.Path, node.PK)
		if err != nil {
			return errors.Wrapf(err,
				"failed to update descendant paths for node with path %s and PK %d",
				node.Path, node.PK,
			)
		}
	}

	err = q.UpdateNode(
		ctx, node.Title, node.Path, node.Depth, node.Numchild, node.UrlPath, node.Slug, int64(node.StatusFlags), node.PageID, node.ContentType, node.LatestRevisionID, node.PK,
	)
	if err != nil {
		return errors.Wrap(err, "failed to update node")
	}

	return SignalNodeUpdated.Send(&PageNodeSignal{
		BaseSignal: BaseSignal{
			Querier: q,
			Ctx:     ctx,
		},
		Node:   node,
		PageID: node.PageID,
	})
}

// DeleteNode deletes a page node.
func DeleteNode(q models.DBQuerier, ctx context.Context, id int64, path string, depth int64) error { //, newParent *models.PageNode) error {
	if depth == 0 {
		return ErrPageIsRoot
	}

	var parentPath, err = ancestorPath(
		path, 1,
	)
	if err != nil {
		return err
	}

	parent, err := q.GetNodeByPath(
		ctx, parentPath,
	)
	if err != nil {
		return err
	}

	prepped, err := PrepareQuerySet(
		ctx, q.DB(),
	)
	if prepped != nil {
		defer prepped.Close()
	}
	if err != nil {
		return err
	}

	tx, err := prepped.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var queries = prepped.WithTx(tx)

	var descendants []models.PageNode
	descendants, err = queries.GetDescendants(
		ctx, path, depth-1, models.StatusFlagNone, 0, 1000,
	)
	if err != nil {
		return err
	}

	var ids = make([]int64, len(descendants))
	for i, descendant := range descendants {
		if err = SignalNodeBeforeDelete.Send(&PageNodeSignal{
			BaseSignal: BaseSignal{
				Querier: q,
				Ctx:     ctx,
			},
			Node:   &descendant,
			PageID: id,
		}); err != nil {
			return err
		}
		ids[i] = descendant.PK
	}

	err = queries.DeleteNodes(ctx, ids)
	if err != nil {
		return err
	}

	parent, err = queries.DecrementNumChild(ctx, parent.PK)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// MoveNode moves a node to a new parent.
//
// The node and new parent paths must not be empty or equal.
//
// The new parent must not be a descendant of the node.
//
// This function will update the url paths of all descendants, as well as the tree paths of the node and its descendants.
func MoveNode(q models.DBQuerier, ctx context.Context, node *models.PageNode, newParent *models.PageNode) error {
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

	prepped, err := PrepareQuerySet(ctx, q.DB())
	if prepped != nil {
		defer prepped.Close()
	}
	if err != nil {
		return errors.Wrap(err, "failed to prepare query set")
	}

	tx, err := prepped.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}

	var queries = prepped.WithTx(tx)
	defer tx.Rollback()

	oldParentPath, err := ancestorPath(node.Path, 1)
	if err != nil {
		return errors.Wrap(err, "failed to get old parent path")
	}

	oldParent, err := queries.GetNodeByPath(ctx, oldParentPath)
	if err != nil {
		return errors.Wrap(err, "failed to get old parent node")
	}

	nodes, err := queries.GetDescendants(ctx, node.Path, node.Depth-1, models.StatusFlagNone, 0, 1000)
	if err != nil {
		return errors.Wrap(err, "failed to get descendants")
	}

	var nodesPtr = make([]*models.PageNode, len(nodes)+1)
	nodesPtr[0] = node

	for _, descendant := range nodes {
		descendant := descendant
		descendant.Path = newParent.Path + descendant.Path[node.Depth*STEP_LEN:]
		descendant.Depth = (newParent.Depth + descendant.Depth + 1) - node.Depth
		// descendant.UrlPath = path.Join(newParent.UrlPath, descendant.Slug)
		// nodesPtr[i+1] = &descendant

		if err = queries.UpdateNodePathAndDepth(ctx, descendant.Path, descendant.Depth, descendant.PK); err != nil {
			return errors.Wrap(err, "failed to update descendant")
		}
	}

	// Update url paths of descendants
	var newPath, oldPath = node.SetUrlPath(newParent)
	node.Path = newParent.Path + buildPathPart(int64(
		newParent.Numchild,
	))
	node.Depth = newParent.Depth + 1

	if err = queries.UpdateNode(ctx, node.Title, node.Path, node.Depth, node.Numchild, newPath, node.Slug, int64(node.StatusFlags), node.PageID, node.ContentType, node.LatestRevisionID, node.PK); err != nil {
		return errors.Wrap(err, "failed to update node")
	}

	if err = queries.UpdateDescendantPaths(ctx, oldPath, newPath, node.Path, node.PK); err != nil {
		return errors.Wrap(err, "failed to update descendant paths")
	}

	*newParent, err = queries.IncrementNumChild(ctx, newParent.PK)
	if err != nil {
		return errors.Wrap(err, "failed to increment new parent numchild")
	}

	_, err = queries.DecrementNumChild(ctx, oldParent.PK)
	if err != nil {
		return errors.Wrap(err, "failed to decrement old parent numchild")
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	return SignalNodeMoved.Send(&PageMovedSignal{
		BaseSignal: BaseSignal{
			Querier: q,
			Ctx:     ctx,
		},
		Node:      node,
		Nodes:     nodesPtr,
		OldParent: &oldParent,
		NewParent: newParent,
	})
}

// PublishNode will set the published flag on the node
// and update it accordingly in the database.
func PublishNode(q models.Querier, ctx context.Context, node *models.PageNode) error {
	if node.Path == "" {
		return fmt.Errorf("node path must not be empty")
	}

	if node.StatusFlags.Is(models.StatusFlagPublished) {
		return nil
	}

	node.StatusFlags |= models.StatusFlagPublished
	return q.UpdateNodeStatusFlags(ctx, int64(models.StatusFlagPublished), node.PK)
}

// UnpublishNode will unset the published flag on the node
// and update it accordingly in the database.
//
// If unpublishChildren is true, it will also unpublish all descendants.
func UnpublishNode(q models.DBQuerier, ctx context.Context, node *models.PageNode, unpublishChildren bool) error {
	if node.Path == "" {
		return fmt.Errorf("node path must not be empty")
	}

	prepped, err := PrepareQuerySet(ctx, q.DB())
	if prepped != nil {
		defer prepped.Close()
	}
	if err != nil {
		return err
	}

	tx, err := prepped.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	var queries = prepped.WithTx(tx)

	if node.StatusFlags.Is(models.StatusFlagPublished) {
		node.StatusFlags &^= models.StatusFlagPublished
	}

	var nodes []*models.PageNode = make([]*models.PageNode, 1)
	if unpublishChildren {
		descendants, err := queries.GetDescendants(ctx, node.Path, node.Depth, models.StatusFlagNone, 0, 1000)
		if err != nil {
			return err
		}

		nodes = make([]*models.PageNode, len(descendants)+1)
		for i := range descendants {
			var d = descendants[i]
			if d.StatusFlags.Is(models.StatusFlagPublished) {
				d.StatusFlags &^= models.StatusFlagPublished
			}
			nodes[i] = &d
		}
	}

	nodes[len(nodes)-1] = node

	if err := queries.UpdateNodes(ctx, nodes); err != nil {
		return err
	}

	return tx.Commit()
}

// ParentNode returns the parent node of the given node.
func ParentNode(q models.Querier, ctx context.Context, path string, depth int) (v models.PageNode, err error) {
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
	return q.GetNodeByPath(
		ctx, parentPath,
	)
}

// AncestorNodes returns the ancestor nodes of the given node.
//
// The path is a PageNode.Path, the depth is the depth of the page.
func AncestorNodes(q models.Querier, ctx context.Context, p string, depth int) ([]models.PageNode, error) {
	var paths = make([]string, depth)
	for i := 1; i < int(depth); i++ {
		var path, err = ancestorPath(
			p, int64(i),
		)
		if err != nil {
			return nil, err
		}
		paths[i] = path
	}
	return q.GetNodesForPaths(
		ctx, paths,
	)
}
