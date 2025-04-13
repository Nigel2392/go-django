package models_mysql

import (
	"context"
	"errors"
	"fmt"
	"strings"

	models "github.com/Nigel2392/go-django/src/contrib/pages/page_models"
	django_models "github.com/Nigel2392/go-django/src/models"
)

const updateNodes = `INSERT INTO PageNode (
	id, title, path, depth, numchild, url_path, slug, status_flags, page_id, content_type, latest_revision_id
) VALUES %REPLACE% ON DUPLICATE KEY UPDATE
	id = VALUES(id),
	title = VALUES(title),
	path = VALUES(path),
	depth = VALUES(depth),
	numchild = VALUES(numchild),
	url_path = VALUES(url_path),
	slug = VALUES(slug),
	status_flags = VALUES(status_flags),
	page_id = VALUES(page_id),
	content_type = VALUES(content_type),
	latest_revision_id = VALUES(latest_revision_id)
`

func (q *Queries) UpdateNodes(ctx context.Context, nodes []*models.PageNode) error {
	var values = make([]interface{}, 0, len(nodes)*8)
	var replaceStrings = make([]string, 0, len(nodes))
	for _, node := range nodes {
		replaceStrings = append(replaceStrings, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
		values = append(values,
			node.PK,
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
		)
	}
	query := updateNodes
	if len(replaceStrings) > 0 {
		query = strings.Replace(query, "%REPLACE%", strings.Join(replaceStrings, ","), 1)
	} else {
		return errors.New("no nodes provided to update")
	}

	_, err := q.exec(ctx, nil, query, values...)
	return err
}

const updateDescendantPaths = `-- name: UpdateDescendantPaths :exec
UPDATE PageNode
SET url_path = CONCAT(?, SUBSTRING(url_path, LENGTH(?) + 1))
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
WHERE id = ?
`

func (q *Queries) IncrementNumChild(ctx context.Context, id int64) (models.PageNode, error) {
	_, err := q.exec(ctx, q.incrementNumChildStmt, incrementNumChild, id)
	var n models.PageNode
	if err != nil {
		return n, err
	}

	return q.GetNodeByID(ctx, id)
}

const decrementNumChild = `-- name: DecrementNumChild :exec
UPDATE PageNode
SET numchild = numchild - 1
WHERE id = ?
`

func (q *Queries) DecrementNumChild(ctx context.Context, id int64) (models.PageNode, error) {
	_, err := q.exec(ctx, q.decrementNumChildStmt, decrementNumChild, id)
	var n models.PageNode
	if err != nil {
		return n, err
	}
	return q.GetNodeByID(ctx, id)
}

const allNodes = `-- name: AllNodes :many
SELECT id, title, path, depth, numchild, url_path, slug, status_flags, page_id, content_type, latest_revision_id, created_at, updated_at
FROM     PageNode
WHERE    status_flags & ? = ?
ORDER BY %s
LIMIT    ?
OFFSET   ?
`

func (q *Queries) AllNodes(ctx context.Context, statusFlags models.StatusFlag, offset int32, limit int32, orderings ...string) ([]models.PageNode, error) {

	var orderer = django_models.Orderer{
		Fields:   orderings,
		Validate: models.IsValidField,
		Default:  models.FieldPath,
	}

	ordering, err := orderer.Build()
	if err != nil {
		return nil, err
	}

	var getAllNodes = fmt.Sprintf(allNodes, ordering)
	rows, err := q.query(ctx, nil, getAllNodes,
		statusFlags,
		statusFlags,
		limit,
		offset,
	)
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
