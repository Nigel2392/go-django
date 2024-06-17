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

	node.Path = buildPathPart(0)
	node.Depth = 0

	id, err := q.InsertNode(ctx, node.Title, node.Path, node.Depth, node.Numchild, int64(node.StatusFlags), node.PageID, node.ContentType)
	if err != nil {
		return err
	}

	node.ID = id

	return nil
}

func CreateChildNode(q models.DBQuerier, ctx context.Context, parent, child *models.PageNode) error {

	var tx, err = q.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	var queries = q.WithTx(tx)

	if parent.Path == "" {
		return fmt.Errorf("parent path must not be empty")
	}

	if child.Path != "" {
		return fmt.Errorf("child path must be empty")
	}

	child.Path = parent.Path + buildPathPart(parent.Numchild)
	child.Depth = parent.Depth + 1

	var id int64
	id, err = queries.InsertNode(ctx, child.Title, child.Path, child.Depth, child.Numchild, int64(child.StatusFlags), child.PageID, child.ContentType)
	if err != nil {
		return err
	}
	child.ID = id
	parent.Numchild++
	err = queries.UpdateNode(ctx, parent.Title, parent.Path, parent.Depth, parent.Numchild, int64(parent.StatusFlags), parent.PageID, parent.ContentType, parent.ID)
	if err != nil {
		return err
	}

	return tx.Commit()
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
	err = queries.UpdateNode(ctx, parent.Title, parent.Path, parent.Depth, parent.Numchild, int64(parent.StatusFlags), parent.PageID, parent.ContentType, parent.ID)
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

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil

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
