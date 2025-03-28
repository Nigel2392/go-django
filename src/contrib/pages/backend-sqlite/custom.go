package models_sqlite

import (
	"context"
	"fmt"
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
	   latest_revision_id = ?
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

const updateDescendantPaths = `-- name: UpdateDescendantPaths :exec
UPDATE PageNode
SET url_path = CONCAT(?, SUBSTR(url_path, LENGTH(?) + 1))
WHERE path LIKE CONCAT(?, '%')
AND id <> ?
`

func (q *Queries) UpdateDescendantPaths(ctx context.Context, oldUrlPath, newUrlPath, pageNodePath string, id int64) error {
	_, err := q.exec(ctx, nil, updateDescendantPaths, newUrlPath, oldUrlPath, pageNodePath, id)
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

const allNodes = `-- name: AllNodes :many
SELECT id, title, path, depth, numchild, url_path, slug, status_flags, page_id, content_type, latest_revision_id, created_at, updated_at
FROM     PageNode
WHERE    status_flags & ?1 = ?1
ORDER BY %s
LIMIT    ?3
OFFSET   ?2`

func (q *Queries) AllNodes(ctx context.Context, statusFlags models.StatusFlag, offset int32, limit int32, orderings ...string) ([]models.PageNode, error) {

	var b strings.Builder
	for i, ordering := range orderings {
		var ord = "ASC"
		if strings.HasPrefix(ordering, "-") {
			ord = "DESC"
			ordering = strings.TrimPrefix(ordering, "-")
		}

		if ordering == "" {
			return nil, fmt.Errorf("ordering field cannot be empty")
		}

		if !models.IsValidField(ordering) {
			return nil, fmt.Errorf("invalid ordering field %s, must be one of %v", ordering, models.ValidFields)
		}

		b.WriteString(ordering)
		b.WriteString(" ")
		b.WriteString(ord)

		if i < len(orderings)-1 {
			b.WriteString(", ")
		}
	}

	if b.Len() == 0 {
		b.WriteString(models.FieldPath)
		b.WriteString(" ASC")
	}

	var getAllNodes = fmt.Sprintf(allNodes, b.String())
	rows, err := q.query(ctx, nil, getAllNodes, statusFlags, offset, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []models.PageNode
	for rows.Next() {
		var i models.PageNode
		if err := rows.Scan(
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
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
