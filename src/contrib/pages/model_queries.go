package pages

import (
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

func (qs *PageQuerySet) queryNodes() ([]*PageNode, error) {
	var rows, err = qs.All()
	if err != nil {
		return nil, fmt.Errorf("failed to query nodes: %w", err)
	}
	return rowsToNodes(rows), nil
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

func (qs *PageQuerySet) incrementNumChild(id int64) (*PageNode, error) {

	var ct, err = qs.
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

	return qs.GetNodeByID(id)
}

func (qs *PageQuerySet) decrementNumChild(id int64) (*PageNode, error) {

	var ct, err = qs.
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
	return qs.GetNodeByID(id)
}

func (qs *PageQuerySet) AllNodes(statusFlags StatusFlag, offset int32, limit int32, orderings ...string) ([]*PageNode, error) {
	return qs.Filter("StatusFlags__bitand", statusFlags).
		OrderBy(orderings...).
		Limit(int(limit)).
		Offset(int(offset)).
		queryNodes()
}

func (qs *PageQuerySet) deleteNodes(id []int64) error {

	if len(id) == 1 {
		qs = qs.Filter("PK", id[0])
	} else if len(id) > 1 {
		qs = qs.Filter("PK__in", id)
	}

	var deleted, err = qs.Delete()
	if err != nil {
		return err
	}
	if deleted == 0 {
		return errors.NoRows
	}
	return err
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
		queryNodes()
}

func (qs *PageQuerySet) GetNodesByIDs(id []int64) ([]*PageNode, error) {
	return qs.Filter("PK__in", id).queryNodes()
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
