package pages

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/Nigel2392/django/contrib/pages/models"
	"github.com/gosimple/slug"
	"github.com/pkg/errors"
)

func setNodeSlug(node *models.PageNode) *models.PageNode {
	node.Slug = strings.TrimSpace(node.Slug)
	if node.Slug == "" {
		node.Slug = slug.Make(node.Title)
	} else {
		node.Slug = slug.Make(node.Slug)
	}

	node.UrlPath = path.Join(node.UrlPath, node.Slug)
	return node
}

func CreateRootNode(q models.Querier, ctx context.Context, node *models.PageNode) error {
	if node.Path != "" {
		return fmt.Errorf("node path must be empty")
	}

	previousRootNodeCount, err := q.CountRootNodes(ctx)
	if err != nil {
		return err
	}

	node.Path = buildPathPart(previousRootNodeCount)
	if node.Title == "" {
		return fmt.Errorf("node title must not be empty")
	}

	node = setNodeSlug(node)

	var urlPath = node.Slug
	if !strings.HasPrefix(urlPath, "/") {
		urlPath = "/" + urlPath
	}
	node.UrlPath = urlPath
	node.Depth = 0

	id, err := q.InsertNode(ctx, node.Title, node.Path, node.Depth, node.Numchild, node.UrlPath, node.Slug, int64(node.StatusFlags), node.PageID, node.ContentType)
	if err != nil {
		return err
	}

	node.PK = id

	return SignalRootCreated.Send(&PageSignal{
		BaseSignal: BaseSignal{
			Querier: q,
			Ctx:     ctx,
		},
		Node: node,
	})
}

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

	child = setNodeSlug(child)

	child.Path = parent.Path + buildPathPart(parent.Numchild)
	child.UrlPath = path.Join(parent.UrlPath, child.Slug)
	child.Depth = parent.Depth + 1
	child.PK, err = queries.InsertNode(ctx, child.Title, child.Path, child.Depth, child.Numchild, child.UrlPath, child.Slug, int64(child.StatusFlags), child.PageID, child.ContentType)
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

	return SignalChildCreated.Send(&PageSignal{
		BaseSignal: BaseSignal{
			Querier: q,
			Ctx:     ctx,
		},
		Node:   child,
		PageID: parent.PageID,
	})
}

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

	nodes, err := queries.GetDescendants(ctx, node.Path, node.Depth-1, 1000, 0)
	if err != nil {
		return errors.Wrap(err, "failed to get descendants")
	}

	var nodesPtr = make([]*models.PageNode, len(nodes)+1)
	nodesPtr[0] = node
	for i, descendant := range nodes {
		descendant := descendant
		descendant.Path = newParent.Path + descendant.Path[node.Depth*STEP_LEN:]
		descendant.Depth = (newParent.Depth + descendant.Depth + 1) - node.Depth
		descendant.UrlPath = path.Join(newParent.UrlPath, descendant.Slug)
		nodesPtr[i+1] = &descendant

		if err = queries.UpdateNodePathAndDepth(ctx, descendant.Path, descendant.Depth, descendant.PK); err != nil {
			return errors.Wrap(err, "failed to update descendant")
		}
	}

	if err = queries.UpdateNodes(ctx, nodesPtr); err != nil {
		return errors.Wrap(err, "failed to update descendants")
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

func UpdateNode(q models.DBQuerier, ctx context.Context, node *models.PageNode) error {
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

	node = setNodeSlug(node)

	var err = q.UpdateNode(
		ctx, node.Title, node.Path, node.Depth, node.Numchild, node.UrlPath, node.Slug, int64(node.StatusFlags), node.PageID, node.ContentType, node.PK,
	)
	if err != nil {
		return err
	}

	return SignalNodeUpdated.Send(&PageSignal{
		BaseSignal: BaseSignal{
			Querier: q,
			Ctx:     ctx,
		},
		Node:   node,
		PageID: node.PageID,
	})
}

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
		ctx, path, depth-1, 1000, 0,
	)
	if err != nil {
		return err
	}

	var ids = make([]int64, len(descendants))
	for i, descendant := range descendants {
		if err = SignalNodeBeforeDelete.Send(&PageSignal{
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
