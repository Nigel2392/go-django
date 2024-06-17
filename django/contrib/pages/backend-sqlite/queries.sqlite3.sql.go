package models_sqlite

import (
	"context"
	"strings"

	"github.com/Nigel2392/django/contrib/pages/models"
)

const allNodes = `-- name: AllNodes :many
SELECT id, title, path, depth, numchild, url_path, status_flags, page_id, content_type, created_at, updated_at
FROM     PageNode
ORDER BY path ASC
LIMIT    ?
OFFSET   ?
`

func (q *Queries) AllNodes(ctx context.Context, limit int32, offset int32) ([]models.PageNode, error) {
	rows, err := q.query(ctx, q.allNodesStmt, allNodes, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []models.PageNode
	for rows.Next() {
		var i models.PageNode
		if err := rows.Scan(
			&i.ID,
			&i.Title,
			&i.Path,
			&i.Depth,
			&i.Numchild,
			&i.UrlPath,
			&i.StatusFlags,
			&i.PageID,
			&i.ContentType,
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

const countNodes = `-- name: CountNodes :one
SELECT COUNT(*)
FROM   PageNode
`

func (q *Queries) CountNodes(ctx context.Context) (int64, error) {
	row := q.queryRow(ctx, q.countNodesStmt, countNodes)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const countRootNodes = `-- name: CountRootNodes :one
SELECT COUNT(*)
FROM   PageNode
WHERE  depth = 0
`

func (q *Queries) CountRootNodes(ctx context.Context) (int64, error) {
	row := q.queryRow(ctx, q.countRootNodesStmt, countRootNodes)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const incrementNumChild = `-- name: IncrementNumChild :exec
UPDATE PageNode
SET numchild = numchild + 1
WHERE path = ?1 AND depth = ?2
RETURNING id, title, path, depth, numchild, url_path, status_flags, page_id, content_type, created_at, updated_at
`

func (q *Queries) IncrementNumChild(ctx context.Context, path string, depth int64) (models.PageNode, error) {
	var row = q.queryRow(ctx, q.incrementNumChildStmt, incrementNumChild, path, depth)
	var i models.PageNode
	var rowErr = row.Err()
	if rowErr != nil {
		return i, rowErr
	}

	err := row.Scan(
		&i.ID,
		&i.Title,
		&i.Path,
		&i.Depth,
		&i.Numchild,
		&i.UrlPath,
		&i.StatusFlags,
		&i.PageID,
		&i.ContentType,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const decrementNumChild = `-- name: DecrementNumChild :exec
UPDATE PageNode
SET numchild = numchild - 1
WHERE path = ?1 AND depth = ?2
RETURNING id, title, path, depth, numchild, url_path, status_flags, page_id, content_type, created_at, updated_at
`

func (q *Queries) DecrementNumChild(ctx context.Context, path string, depth int64) (models.PageNode, error) {
	var row = q.queryRow(ctx, q.decrementNumChildStmt, decrementNumChild, path, depth)
	var i models.PageNode
	var rowErr = row.Err()
	if rowErr != nil {
		return i, rowErr
	}

	err := row.Scan(
		&i.ID,
		&i.Title,
		&i.Path,
		&i.Depth,
		&i.Numchild,
		&i.UrlPath,
		&i.StatusFlags,
		&i.PageID,
		&i.ContentType,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const deleteDescendants = `-- name: DeleteDescendants :exec
DELETE FROM PageNode
WHERE path LIKE CONCAT(?1, '%') AND depth > ?2
`

func (q *Queries) DeleteDescendants(ctx context.Context, path interface{}, depth int64) error {
	_, err := q.exec(ctx, q.deleteDescendantsStmt, deleteDescendants, path, depth)
	return err
}

const deleteNode = `-- name: DeleteNode :exec
DELETE FROM PageNode
WHERE id = ?1
`

func (q *Queries) DeleteNode(ctx context.Context, id int64) error {
	_, err := q.exec(ctx, q.deleteNodeStmt, deleteNode, id)
	return err
}

const deleteNodes = `-- name: DeleteNodes :exec
DELETE FROM PageNode
WHERE id IN (/*SLICE:id*/?)
`

func (q *Queries) DeleteNodes(ctx context.Context, id []int64) error {
	query := deleteNodes
	var queryParams []interface{}
	if len(id) > 0 {
		for _, v := range id {
			queryParams = append(queryParams, v)
		}
		query = strings.Replace(query, "/*SLICE:id*/?", strings.Repeat(",?", len(id))[1:], 1)
	} else {
		query = strings.Replace(query, "/*SLICE:id*/?", "NULL", 1)
	}
	_, err := q.exec(ctx, nil, query, queryParams...)
	return err
}

const getChildNodes = `-- name: GetChildNodes :many
SELECT   id, title, path, depth, numchild, url_path, status_flags, page_id, content_type, created_at, updated_at
FROM     PageNode
WHERE    path LIKE CONCAT(?1, '%') AND depth = ?2 + 1
`

func (q *Queries) GetChildNodes(ctx context.Context, path interface{}, depth interface{}) ([]models.PageNode, error) {
	rows, err := q.query(ctx, q.getChildNodesStmt, getChildNodes, path, depth)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []models.PageNode
	for rows.Next() {
		var i models.PageNode
		if err := rows.Scan(
			&i.ID,
			&i.Title,
			&i.Path,
			&i.Depth,
			&i.Numchild,
			&i.UrlPath,
			&i.StatusFlags,
			&i.PageID,
			&i.ContentType,
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

const getDescendants = `-- name: GetDescendants :many
SELECT   id, title, path, depth, numchild, url_path, status_flags, page_id, content_type, created_at, updated_at
FROM     PageNode
WHERE    path LIKE CONCAT(?1, '%') AND depth > ?2
`

func (q *Queries) GetDescendants(ctx context.Context, path interface{}, depth int64) ([]models.PageNode, error) {
	rows, err := q.query(ctx, q.getDescendantsStmt, getDescendants, path, depth)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []models.PageNode
	for rows.Next() {
		var i models.PageNode
		if err := rows.Scan(
			&i.ID,
			&i.Title,
			&i.Path,
			&i.Depth,
			&i.Numchild,
			&i.UrlPath,
			&i.StatusFlags,
			&i.PageID,
			&i.ContentType,
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

const getNodeByID = `-- name: GetNodeByID :one
SELECT   id, title, path, depth, numchild, url_path, status_flags, page_id, content_type, created_at, updated_at
FROM     PageNode
WHERE    id = ?1
`

func (q *Queries) GetNodeByID(ctx context.Context, id int64) (models.PageNode, error) {
	row := q.queryRow(ctx, q.getNodeByIDStmt, getNodeByID, id)
	var i models.PageNode
	err := row.Scan(
		&i.ID,
		&i.Title,
		&i.Path,
		&i.Depth,
		&i.Numchild,
		&i.UrlPath,
		&i.StatusFlags,
		&i.PageID,
		&i.ContentType,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const getNodeByPath = `-- name: GetNodeByPath :one
SELECT   id, title, path, depth, numchild, url_path, status_flags, page_id, content_type, created_at, updated_at
FROM     PageNode
WHERE    path = ?1
`

func (q *Queries) GetNodeByPath(ctx context.Context, path string) (models.PageNode, error) {
	row := q.queryRow(ctx, q.getNodeByPathStmt, getNodeByPath, path)
	var i models.PageNode
	err := row.Scan(
		&i.ID,
		&i.Title,
		&i.Path,
		&i.Depth,
		&i.Numchild,
		&i.UrlPath,
		&i.StatusFlags,
		&i.PageID,
		&i.ContentType,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const getNodesByIDs = `-- name: GetNodesByIDs :many
SELECT   id, title, path, depth, numchild, url_path, status_flags, page_id, content_type, created_at, updated_at
FROM     PageNode
WHERE    id IN (/*SLICE:id*/?)
`

func (q *Queries) GetNodesByIDs(ctx context.Context, id []int64) ([]models.PageNode, error) {
	query := getNodesByIDs
	var queryParams []interface{}
	if len(id) > 0 {
		for _, v := range id {
			queryParams = append(queryParams, v)
		}
		query = strings.Replace(query, "/*SLICE:id*/?", strings.Repeat(",?", len(id))[1:], 1)
	} else {
		query = strings.Replace(query, "/*SLICE:id*/?", "NULL", 1)
	}
	rows, err := q.query(ctx, nil, query, queryParams...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []models.PageNode
	for rows.Next() {
		var i models.PageNode
		if err := rows.Scan(
			&i.ID,
			&i.Title,
			&i.Path,
			&i.Depth,
			&i.Numchild,
			&i.UrlPath,
			&i.StatusFlags,
			&i.PageID,
			&i.ContentType,
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

const getNodesByPageIDs = `-- name: GetNodesByPageIDs :many
SELECT   id, title, path, depth, numchild, url_path, status_flags, page_id, content_type, created_at, updated_at
FROM     PageNode
WHERE    page_id IN (/*SLICE:page_id*/?)
`

func (q *Queries) GetNodesByPageIDs(ctx context.Context, pageID []int64) ([]models.PageNode, error) {
	query := getNodesByPageIDs
	var queryParams []interface{}
	if len(pageID) > 0 {
		for _, v := range pageID {
			queryParams = append(queryParams, v)
		}
		query = strings.Replace(query, "/*SLICE:page_id*/?", strings.Repeat(",?", len(pageID))[1:], 1)
	} else {
		query = strings.Replace(query, "/*SLICE:page_id*/?", "NULL", 1)
	}
	rows, err := q.query(ctx, nil, query, queryParams...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []models.PageNode
	for rows.Next() {
		var i models.PageNode
		if err := rows.Scan(
			&i.ID,
			&i.Title,
			&i.Path,
			&i.Depth,
			&i.Numchild,
			&i.UrlPath,
			&i.StatusFlags,
			&i.PageID,
			&i.ContentType,
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

const getNodesByTypeHash = `-- name: GetNodesByTypeHash :many
SELECT   id, title, path, depth, numchild, url_path, status_flags, page_id, content_type, created_at, updated_at
FROM     PageNode
WHERE    content_type = ?1
`

func (q *Queries) GetNodesByTypeHash(ctx context.Context, contentType string) ([]models.PageNode, error) {
	rows, err := q.query(ctx, q.getNodesByTypeHashStmt, getNodesByTypeHash, contentType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []models.PageNode
	for rows.Next() {
		var i models.PageNode
		if err := rows.Scan(
			&i.ID,
			&i.Title,
			&i.Path,
			&i.Depth,
			&i.Numchild,
			&i.UrlPath,
			&i.StatusFlags,
			&i.PageID,
			&i.ContentType,
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

const getNodesByTypeHashes = `-- name: GetNodesByTypeHashes :many
SELECT   id, title, path, depth, numchild, url_path, status_flags, page_id, content_type, created_at, updated_at
FROM     PageNode
WHERE    content_type IN (/*SLICE:content_type*/?)
`

func (q *Queries) GetNodesByTypeHashes(ctx context.Context, contentType []string) ([]models.PageNode, error) {
	query := getNodesByTypeHashes
	var queryParams []interface{}
	if len(contentType) > 0 {
		for _, v := range contentType {
			queryParams = append(queryParams, v)
		}
		query = strings.Replace(query, "/*SLICE:content_type*/?", strings.Repeat(",?", len(contentType))[1:], 1)
	} else {
		query = strings.Replace(query, "/*SLICE:content_type*/?", "NULL", 1)
	}
	rows, err := q.query(ctx, nil, query, queryParams...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []models.PageNode
	for rows.Next() {
		var i models.PageNode
		if err := rows.Scan(
			&i.ID,
			&i.Title,
			&i.Path,
			&i.Depth,
			&i.Numchild,
			&i.UrlPath,
			&i.StatusFlags,
			&i.PageID,
			&i.ContentType,
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

const getNodesForPaths = `-- name: GetNodesForPaths :many
SELECT   id, title, path, depth, numchild, url_path, status_flags, page_id, content_type, created_at, updated_at
FROM     PageNode
WHERE    path IN (/*SLICE:path*/?)
`

func (q *Queries) GetNodesForPaths(ctx context.Context, path []string) ([]models.PageNode, error) {
	query := getNodesForPaths
	var queryParams []interface{}
	if len(path) > 0 {
		for _, v := range path {
			queryParams = append(queryParams, v)
		}
		query = strings.Replace(query, "/*SLICE:path*/?", strings.Repeat(",?", len(path))[1:], 1)
	} else {
		query = strings.Replace(query, "/*SLICE:path*/?", "NULL", 1)
	}
	rows, err := q.query(ctx, nil, query, queryParams...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []models.PageNode
	for rows.Next() {
		var i models.PageNode
		if err := rows.Scan(
			&i.ID,
			&i.Title,
			&i.Path,
			&i.Depth,
			&i.Numchild,
			&i.UrlPath,
			&i.StatusFlags,
			&i.PageID,
			&i.ContentType,
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

const insertNode = `-- name: InsertNode :execlastid
INSERT INTO PageNode (
    title,
    path,
    depth,
    numchild,
    url_path,
    status_flags,
    page_id,
    content_type
) VALUES (
    ?1,
    ?2,
    ?3,
    ?4,
    ?5,
    ?6,
    ?7,
    ?8
)
`

func (q *Queries) InsertNode(ctx context.Context, title string, path string, depth int64, numchild int64, urlPath string, statusFlags int64, pageID int64, contentType string) (int64, error) {
	result, err := q.exec(ctx, q.insertNodeStmt, insertNode,
		title,
		path,
		depth,
		numchild,
		urlPath,
		statusFlags,
		pageID,
		contentType,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

const updateNode = `-- name: UpdateNode :exec
UPDATE PageNode
SET title = ?1,
    path = ?2,
    depth = ?3, 
    numchild = ?4,
    url_path = ?5,
    status_flags = ?6, 
    page_id = ?7, 
    content_type = ?8,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?9
`

func (q *Queries) UpdateNode(ctx context.Context, title string, path string, depth int64, numchild int64, urlPath string, statusFlags int64, pageID int64, contentType string, iD int64) error {
	_, err := q.exec(ctx, q.updateNodeStmt, updateNode,
		title,
		path,
		depth,
		numchild,
		urlPath,
		statusFlags,
		pageID,
		contentType,
		iD,
	)
	return err
}

const updateNodes = `UPDATE PageNode
SET    title = ?,
	   path = ?,
	   depth = ?,
	   numchild = ?,
	   url_path = ?,
	   status_flags = ?,
	   page_id = ?,
	   content_type = ?,
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
			node.StatusFlags,
			node.PageID,
			node.ContentType,
			node.ID,
		)
	}

	_, err := q.exec(ctx, nil, query.String(), values...)
	return err

}

const updateNodePathAndDepth = `-- name: UpdateNodePathAndDepth :exec
UPDATE   PageNode
SET      path = ?1, depth = ?2
WHERE    id = ?3
`

func (q *Queries) UpdateNodePathAndDepth(ctx context.Context, path string, depth int64, iD int64) error {
	_, err := q.exec(ctx, q.updateNodePathAndDepthStmt, updateNodePathAndDepth, path, depth, iD)
	return err
}

const updateNodeStatusFlags = `-- name: UpdateNodeStatusFlags :exec
UPDATE   PageNode
SET      status_flags = ?1
WHERE    id = ?2
`

func (q *Queries) UpdateNodeStatusFlags(ctx context.Context, statusFlags int64, iD int64) error {
	_, err := q.exec(ctx, q.updateNodeStatusFlagsStmt, updateNodeStatusFlags, statusFlags, iD)
	return err
}
