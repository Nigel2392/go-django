package pages

import (
	"context"
	"fmt"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
)

func rowsToNodes(rows queries.Rows[*PageNode]) []*PageNode {
	var nodes = make([]*PageNode, 0, len(rows))
	for obj := range rows.Objects() {
		nodes = append(nodes, obj)
	}
	return nodes
}

func updateNodes(ctx context.Context, nodes []*PageNode) error {
	var updated, err = queries.GetQuerySet(&PageNode{}).WithContext(ctx).
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

func updateDescendantPaths(ctx context.Context, oldUrlPath, newUrlPath, pageNodePath string, id int64) error {
	var updated, err = queries.GetQuerySet(&PageNode{}).WithContext(ctx).
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
	if err != nil {
		return fmt.Errorf("failed to update descendant paths: %w", err)
	}
	if updated == 0 {
		return errors.New(errors.CodeNoChanges, "no descendant paths were updated")
	}
	return err
}

func incrementNumChild(ctx context.Context, id int64) (*PageNode, error) {

	var ct, err = queries.GetQuerySet(&PageNode{}).WithContext(ctx).
		Select("Numchild").
		Filter("PK", id).
		ExplicitSave().
		Update(
			&PageNode{},
			expr.As("Numchild", expr.Logical("Numchild").ADD(1)),
		)
	if err != nil {
		return nil, fmt.Errorf("failed to increment numchild: %w", err)
	}
	if ct == 0 {
		return nil, fmt.Errorf("no nodes were updated for id %d", id)
	}

	return GetNodeByID(ctx, id)
}

func decrementNumChild(ctx context.Context, id int64) (*PageNode, error) {
	var ct, err = queries.GetQuerySet(&PageNode{}).WithContext(ctx).
		Select("Numchild").
		Filter("PK", id).
		ExplicitSave().
		Update(
			&PageNode{},
			expr.As("Numchild", expr.Logical("Numchild").SUB(1)),
		)
	if err != nil {
		return nil, fmt.Errorf("failed to decrement numchild: %w", err)
	}
	if ct == 0 {
		return nil, fmt.Errorf("no nodes were updated for id %d", id)
	}
	return GetNodeByID(ctx, id)
}

func AllNodes(ctx context.Context, statusFlags StatusFlag, offset int32, limit int32, orderings ...string) ([]*PageNode, error) {
	var nodes, err = queries.GetQuerySet(&PageNode{}).WithContext(ctx).
		Filter("StatusFlags__bitand", statusFlags).
		OrderBy(orderings...).
		Limit(int(limit)).
		Offset(int(offset)).
		All()
	if err != nil {
		return nil, err
	}
	return rowsToNodes(nodes), nil
}

func CountNodes(ctx context.Context, statusFlags StatusFlag) (int64, error) {
	var nodesCount, err = queries.GetQuerySet(&PageNode{}).WithContext(ctx).
		Filter("StatusFlags__bitand", statusFlags).
		Count()
	if err != nil {
		return 0, err
	}
	return nodesCount, nil
}

func CountNodesByTypeHash(ctx context.Context, contentType string) (int64, error) {
	var nodesCount, err = queries.GetQuerySet(&PageNode{}).WithContext(ctx).
		Filter("ContentType", contentType).
		Count()
	if err != nil {
		return 0, err
	}
	return nodesCount, nil
}

func CountRootNodes(ctx context.Context, statusFlags StatusFlag) (int64, error) {
	var count, err = queries.GetQuerySet(&PageNode{}).WithContext(ctx).
		Filter("Depth", 0).
		Filter("StatusFlags__bitand", statusFlags).
		Count()
	if err != nil {
		return 0, err
	}
	return count, nil
}

func deleteDescendants(ctx context.Context, path string, depth int64) error {
	var deleted, err = queries.GetQuerySet(&PageNode{}).WithContext(ctx).
		Filter("Path__startswith", path).
		Filter("Depth__gt", depth).
		Delete()
	if err != nil {
		return err
	}
	if deleted == 0 {
		return errors.NoRows
	}
	return err
}

func deleteNode(ctx context.Context, id int64) error {
	var deleted, err = queries.GetQuerySet(&PageNode{}).WithContext(ctx).
		Filter("PK", id).
		Delete()
	if err != nil {
		return err
	}
	if deleted == 0 {
		return errors.NoRows
	}
	return err
}

func deleteNodes(ctx context.Context, id []int64) error {
	var deleted, err = queries.GetQuerySet(&PageNode{}).WithContext(ctx).
		Filter("PK__in", id).
		Delete()
	if err != nil {
		return err
	}
	if deleted == 0 {
		return errors.NoRows
	}
	return err
}

func GetChildNodes(ctx context.Context, node *PageNode, statusFlags StatusFlag, offset int32, limit int32) ([]*PageNode, error) {
	var rows, err = queries.GetQuerySet(&PageNode{}).WithContext(ctx).
		Select("*").
		Filter(
			expr.Expr("Depth", expr.LOOKUP_EXACT, node.Depth+1),
			expr.Expr("Path", expr.LOOKUP_STARTSWITH, node.Path),
			expr.Expr("StatusFlags", expr.LOOKUP_BITAND, statusFlags),
		).
		Limit(int(limit)).
		Offset(int(offset)).
		OrderBy("Path").
		All()
	if err != nil {
		return nil, err
	}

	return rowsToNodes(rows), nil
}

func GetDescendants(ctx context.Context, path string, depth int64, statusFlags StatusFlag, offset int32, limit int32) ([]*PageNode, error) {
	var rows, err = queries.GetQuerySet(&PageNode{}).WithContext(ctx).
		Filter(
			expr.Expr("Path", expr.LOOKUP_STARTSWITH, path),
			expr.Expr("Depth", expr.LOOKUP_GT, depth),
			expr.Expr("StatusFlags", expr.LOOKUP_BITAND, statusFlags),
		).
		Limit(int(limit)).
		Offset(int(offset)).
		OrderBy("Path").
		All()
	if err != nil {
		return nil, err
	}
	return rowsToNodes(rows), nil
}

func GetNodeByID(ctx context.Context, id int64) (*PageNode, error) {
	var row, err = queries.GetQuerySet(&PageNode{}).WithContext(ctx).
		Filter("PK", id).
		Get()
	if err != nil {
		return nil, err
	}
	return row.Object, nil
}

func GetNodeByPath(ctx context.Context, path string) (*PageNode, error) {
	var row, err = queries.GetQuerySet(&PageNode{}).WithContext(ctx).
		Filter("Path", path).
		Get()
	if err != nil {
		return nil, err
	}
	return row.Object, nil
}

func GetNodeBySlug(ctx context.Context, slug string, depth int64, path string) (*PageNode, error) {
	var row, err = queries.GetQuerySet(&PageNode{}).WithContext(ctx).
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

func GetNodesByDepth(ctx context.Context, depth int64, statusFlags StatusFlag, offset int32, limit int32) ([]*PageNode, error) {
	var rows, err = queries.GetQuerySet(&PageNode{}).WithContext(ctx).
		Filter(
			expr.Expr("Depth", expr.LOOKUP_EXACT, depth),
			expr.Expr("StatusFlags", expr.LOOKUP_BITAND, statusFlags),
		).
		Limit(int(limit)).
		Offset(int(offset)).
		OrderBy("Path").
		All()
	if err != nil {
		return nil, err
	}
	return rowsToNodes(rows), nil
}

func GetNodesByIDs(ctx context.Context, id []int64) ([]*PageNode, error) {
	var rows, err = queries.GetQuerySet(&PageNode{}).WithContext(ctx).
		Filter("PK__in", id).
		All()
	if err != nil {
		return nil, err
	}
	return rowsToNodes(rows), nil
}

func GetNodesByPageIDs(ctx context.Context, pageID []int64) ([]*PageNode, error) {
	var rows, err = queries.GetQuerySet(&PageNode{}).WithContext(ctx).
		Filter("PageID__in", pageID).
		All()
	if err != nil {
		return nil, err
	}
	return rowsToNodes(rows), nil
}

func GetNodesByTypeHash(ctx context.Context, contentType string, offset int32, limit int32) ([]*PageNode, error) {
	var rows, err = queries.GetQuerySet(&PageNode{}).WithContext(ctx).
		Filter("ContentType", contentType).
		Limit(int(limit)).
		Offset(int(offset)).
		OrderBy("Path").
		All()
	if err != nil {
		return nil, err
	}
	return rowsToNodes(rows), nil
}

func GetNodesByTypeHashes(ctx context.Context, contentType []string, offset int32, limit int32) ([]*PageNode, error) {
	var rows, err = queries.GetQuerySet(&PageNode{}).WithContext(ctx).
		Filter("ContentType__in", contentType).
		Limit(int(limit)).
		Offset(int(offset)).
		OrderBy("Path").
		All()
	if err != nil {
		return nil, err
	}
	return rowsToNodes(rows), nil
}

func GetNodesForPaths(ctx context.Context, path []string) ([]*PageNode, error) {
	var rows, err = queries.GetQuerySet(&PageNode{}).WithContext(ctx).
		Filter("Path__in", path).
		All()
	if err != nil {
		return nil, err
	}
	return rowsToNodes(rows), nil
}

func insertNode(ctx context.Context, node *PageNode) (int64, error) {
	var err error
	node, err = queries.GetQuerySet(node).WithContext(ctx).ExplicitSave().Create(node)
	if err != nil {
		return 0, err
	}
	return node.PK, nil
}

func updateNode(ctx context.Context, node *PageNode) error {
	var updated, err = queries.GetQuerySet(&PageNode{}).WithContext(ctx).
		Select("PK", "Title", "Path", "Depth", "Numchild", "UrlPath", "Slug", "StatusFlags", "PageID", "ContentType", "LatestRevisionID", "UpdatedAt").
		Filter("PK", node.PK).
		ExplicitSave().
		Update(node)
	if err != nil {
		return err
	}
	if updated == 0 {
		return errors.NoChanges
	}
	return nil
}

func updateNodePathAndDepth(ctx context.Context, path string, depth int64, iD int64) error {
	var updated, err = queries.GetQuerySet(&PageNode{}).WithContext(ctx).
		Select("Path", "Depth").
		Filter("PK", iD).
		ExplicitSave().
		Update(
			&PageNode{
				Path:  path,
				Depth: depth,
			},
		)
	if err != nil {
		return err
	}
	if updated == 0 {
		return errors.NoChanges
	}
	return nil
}

func updateNodeStatusFlags(ctx context.Context, statusFlags int64, iD int64) error {
	var updated, err = queries.GetQuerySet(&PageNode{}).WithContext(ctx).
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
		return errors.NoChanges
	}
	return nil
}
