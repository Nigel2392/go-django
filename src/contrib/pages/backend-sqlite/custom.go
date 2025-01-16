package models_sqlite

import (
	"context"
	"strings"

	models "github.com/Nigel2392/go-django/src/contrib/pages/page_models"
)

const updateNodes = `UPDATE PageNode
SET    title = ?,
	   path = ?,
	   depth = ?,
	   numchild = ?,
	   url_path = ?,
	   slug = ?,
	   status_flags = ?,
	   page_id = ?,
	   content_type = ?,
	   latest_revision_id = ?,
	   updated_at = CURRENT_TIMESTAMP
WHERE  id = ?
`

func (q *Queries) UpdateNodes(ctx context.Context, nodes []*models.PageNode) error {
	var query strings.Builder
	var values = make([]interface{}, 0, len(nodes)*8)
	for i, node := range nodes {
		query.WriteString(updateNodes)
		if i < len(nodes)-1 {
			query.WriteString("; ")
		}
		values = append(values,
			node.Title,
			node.Path,
			node.Depth,
			node.Numchild,
			node.UrlPath,
			node.Slug,
			node.StatusFlags,
			node.PageID,
			node.ContentType,
			node.LatestRevisionID,
			node.PK,
		)
	}

	_, err := q.exec(ctx, nil, query.String(), values...)
	return err

}

const incrementNumChild = `-- name: IncrementNumChild :exec
UPDATE PageNode
SET numchild = numchild + 1
WHERE id = ?1
RETURNING id, title, path, depth, numchild, url_path, slug, status_flags, page_id, content_type, latest_revision_id, created_at, updated_at
`

func (q *Queries) IncrementNumChild(ctx context.Context, id int64) (models.PageNode, error) {
	row := q.queryRow(ctx, q.incrementNumChildStmt, incrementNumChild, id)
	var i models.PageNode
	var rowErr = row.Err()
	if rowErr != nil {
		return i, rowErr
	}
	err := row.Scan(
		&i.PK,
		&i.Title,
		&i.Path,
		&i.Depth,
		&i.Numchild,
		&i.UrlPath,
		&i.Slug,
		&i.StatusFlags,
		&i.PageID,
		&i.ContentType,
		&i.LatestRevisionID,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const decrementNumChild = `-- name: DecrementNumChild :exec
UPDATE PageNode
SET numchild = numchild - 1
WHERE id = ?1
RETURNING id, title, path, depth, numchild, url_path, slug, status_flags, page_id, content_type, latest_revision_id, created_at, updated_at
`

func (q *Queries) DecrementNumChild(ctx context.Context, id int64) (models.PageNode, error) {
	var row = q.queryRow(ctx, q.decrementNumChildStmt, decrementNumChild, id)
	var i models.PageNode
	var rowErr = row.Err()
	if rowErr != nil {
		return i, rowErr
	}

	err := row.Scan(
		&i.PK,
		&i.Title,
		&i.Path,
		&i.Depth,
		&i.Numchild,
		&i.UrlPath,
		&i.Slug,
		&i.StatusFlags,
		&i.PageID,
		&i.ContentType,
		&i.LatestRevisionID,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}
