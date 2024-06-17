package pages

import (
	"context"
	"fmt"

	"github.com/Nigel2392/django/contrib/pages/models"
)

func CreateRootNode(q models.Querier, ctx context.Context, node *models.PageNode) error {
	if node.Path != "" {
		return fmt.Errorf("node path must be empty")
	}

	previousRootNodeCount, err := q.CountRootNodes(ctx)
	if err != nil {
		return err
	}

	node.Path = buildPathPart(previousRootNodeCount)
	node.Depth = 0

	id, err := q.InsertNode(ctx, node.Title, node.Path, node.Depth, node.Numchild, node.Path, int64(node.StatusFlags), node.PageID, node.ContentType)
	if err != nil {
		return err
	}

	node.ID = id

	return SignalRootCreated.Send(&PageSignal{
		Querier: q,
		Ctx:     ctx,
		Node:    node,
	})
}

func CreateChildNode(q models.DBQuerier, ctx context.Context, parent, child *models.PageNode) error {

	var prepped, err = PrepareQuerySet(ctx, q.DB())
	if err != nil {
		return err
	}

	tx, err := prepped.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	var queries = prepped.WithTx(tx)
	defer tx.Rollback()
	defer queries.Close()

	if parent.Path == "" {
		return fmt.Errorf("parent path must not be empty")
	}

	if child.Path != "" {
		return fmt.Errorf("child path must be empty")
	}

	child.Path = parent.Path + buildPathPart(parent.Numchild)
	child.Depth = parent.Depth + 1

	var id int64
	id, err = queries.InsertNode(ctx, child.Title, child.Path, child.Depth, child.Numchild, child.UrlPath, int64(child.StatusFlags), child.PageID, child.ContentType)
	if err != nil {
		return err
	}
	child.ID = id
	parent.Numchild++
	err = queries.UpdateNode(ctx, parent.Title, parent.Path, parent.Depth, parent.Numchild, parent.UrlPath, int64(parent.StatusFlags), parent.PageID, parent.ContentType, parent.ID)
	if err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return SignalChildCreated.Send(&PageSignal{
		Querier: q,
		Ctx:     ctx,
		Node:    child,
		PageID:  parent.PageID,
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

	if node.ID == 0 {
		return fmt.Errorf("node id must not be zero")
	}

	var err = q.UpdateNode(
		ctx, node.Title, node.Path, node.Depth, node.Numchild, node.UrlPath, int64(node.StatusFlags), node.PageID, node.ContentType, node.ID,
	)
	if err != nil {
		return err
	}

	return SignalNodeUpdated.Send(&PageSignal{
		Querier: q,
		Ctx:     ctx,
		Node:    node,
		PageID:  node.PageID,
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

	if err = SignalNodeBeforeDelete.Send(&PageSignal{
		Querier: q,
		Ctx:     ctx,
		Node:    &parent,
		PageID:  id,
	}); err != nil {
		return err
	}

	tx, err := q.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	var queries = q.WithTx(tx)

	err = queries.DeleteNode(ctx, id)
	if err != nil {
		return err
	}

	parent.Numchild--
	err = queries.UpdateNode(ctx, parent.Title, parent.Path, parent.Depth, parent.Numchild, parent.UrlPath, int64(parent.StatusFlags), parent.PageID, parent.ContentType, parent.ID)
	if err != nil {
		return err
	}

	//if newParent != nil {
	//	newParent.Numchild++
	//	err = queries.UpdateNode(ctx, newParent.Title, newParent.Path, newParent.Depth, newParent.Numchild, int64(newParent.StatusFlags), newParent.PageID, newParent.ContentType, newParent.ID)
	//	if err != nil {
	//		return err
	//	}
	//
	//	var descendants []models.PageNode
	//	descendants, err = q.GetDescendants(
	//		ctx, path, depth,
	//	)
	//	if err != nil {
	//		return err
	//	}
	//
	//	for _, descendant := range descendants {
	//		descendant.Path = newParent.Path + descendant.Path[len(path):]
	//		descendant.Depth = newParent.Depth + descendant.Depth - depth
	//		err = queries.UpdateNodePathAndDepth(
	//			ctx, descendant.Path, descendant.Depth, descendant.ID,
	//		)
	//		if err != nil {
	//			return err
	//		}
	//	}
	//
	//	return tx.Commit()
	//
	//}

	return tx.Commit()

	//return queries.DeleteDescendants(
	//	ctx, path, depth,
	//)
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
