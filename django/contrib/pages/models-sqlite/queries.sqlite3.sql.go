package models_sqlite

import (
	"context"
	"strings"

	"github.com/Nigel2392/django/contrib/pages/models"
)

const allNodes = `-- name: AllNodes :many
SELECT id, title, path, depth, numchild, status_flags, page_id, content_type, created_at, updated_at
FROM     PageNode
ORDER BY path ASC
LIMIT    ?2
OFFSET   ?1
`

func (q *Queries) AllNodes(ctx context.Context, nodeOffset int32, nodeLimit int32) ([]models.PageNode, error) {
	rows, err := q.query(ctx, q.allNodesStmt, allNodes, nodeOffset, nodeLimit)
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
SELECT   id, title, path, depth, numchild, status_flags, page_id, content_type, created_at, updated_at
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
SELECT   id, title, path, depth, numchild, status_flags, page_id, content_type, created_at, updated_at
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
SELECT   id, title, path, depth, numchild, status_flags, page_id, content_type, created_at, updated_at
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
		&i.StatusFlags,
		&i.PageID,
		&i.ContentType,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const getNodeByPath = `-- name: GetNodeByPath :one
SELECT   id, title, path, depth, numchild, status_flags, page_id, content_type, created_at, updated_at
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
		&i.StatusFlags,
		&i.PageID,
		&i.ContentType,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const getNodesByIDs = `-- name: GetNodesByIDs :many
SELECT   id, title, path, depth, numchild, status_flags, page_id, content_type, created_at, updated_at
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
SELECT   id, title, path, depth, numchild, status_flags, page_id, content_type, created_at, updated_at
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
SELECT   id, title, path, depth, numchild, status_flags, page_id, content_type, created_at, updated_at
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
SELECT   id, title, path, depth, numchild, status_flags, page_id, content_type, created_at, updated_at
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

const getNodesForPath = `-- name: GetNodesForPaths :many
SELECT   id, title, path, depth, numchild, status_flags, page_id, content_type, created_at, updated_at
FROM     PageNode
WHERE    path IN (/*SLICE:path*/?)
`

func (q *Queries) GetNodesForPaths(ctx context.Context, path []string) ([]models.PageNode, error) {
	query := getNodesForPath
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
    ?7
)
`

func (q *Queries) InsertNode(ctx context.Context, title string, path string, depth int64, numchild int64, statusFlags int64, pageID int64, contentType string) (int64, error) {
	result, err := q.exec(ctx, q.insertNodeStmt, insertNode,
		title,
		path,
		depth,
		numchild,
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
    status_flags = ?5, 
    page_id = ?6, 
    content_type = ?7,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?8
`

func (q *Queries) UpdateNode(ctx context.Context, title string, path string, depth int64, numchild int64, statusFlags int64, pageID int64, contentType string, iD int64) error {
	_, err := q.exec(ctx, q.updateNodeStmt, updateNode,
		title,
		path,
		depth,
		numchild,
		statusFlags,
		pageID,
		contentType,
		iD,
	)
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
