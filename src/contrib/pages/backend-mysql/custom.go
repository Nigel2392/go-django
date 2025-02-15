package models_mysql

import (
	"context"
	"errors"
	"strings"

	models "github.com/Nigel2392/go-django/src/contrib/pages/page_models"
)

const updateNodes = `INSERT INTO PageNode (
	id, title, path, depth, numchild, url_path, slug, status_flags, page_id, content_type, created_at, updated_at
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
	latest_revision_id = VALUES(latest_revision_id),
	updated_at = VALUES(updated_at)
`

func (q *Queries) UpdateNodes(ctx context.Context, nodes []*models.PageNode) error {
	var values = make([]interface{}, 0, len(nodes)*8)
	var replaceStrings = make([]string, 0, len(nodes))
	for _, node := range nodes {
		replaceStrings = append(replaceStrings, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)")
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
