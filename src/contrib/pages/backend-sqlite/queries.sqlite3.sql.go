package models_sqlite

import (
	"context"
	"strings"

	"github.com/Nigel2392/go-django/src/contrib/pages/page_models"
)

const allNodes = `-- name: AllNodes :many
SELECT id, title, path, depth, numchild, url_path, slug, status_flags, page_id, content_type, latest_revision_id, created_at, updated_at
FROM     PageNode
WHERE    status_flags & ?1 = ?1
ORDER BY path ASC
LIMIT    ?3
OFFSET   ?2
`

func (q *Queries) AllNodes(ctx context.Context, statusFlags page_models.StatusFlag, offset int32, limit int32) ([]page_models.PageNode, error) {
	rows, err := q.query(ctx, q.allNodesStmt, allNodes, statusFlags, offset, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []page_models.PageNode
	for rows.Next() {
		var i page_models.PageNode
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

const countNodes = `-- name: CountNodes :one
SELECT COUNT(*)
FROM   PageNode
WHERE status_flags & ?1 = ?1
`

func (q *Queries) CountNodes(ctx context.Context, statusFlags page_models.StatusFlag) (int64, error) {
	row := q.queryRow(ctx, q.countNodesStmt, countNodes, statusFlags)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const countNodesByTypeHash = `-- name: CountNodesByTypeHash :one
SELECT COUNT(*)
FROM   PageNode
WHERE  content_type = ?1
`

func (q *Queries) CountNodesByTypeHash(ctx context.Context, contentType string) (int64, error) {
	row := q.queryRow(ctx, q.countNodesByTypeHashStmt, countNodesByTypeHash, contentType)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const countRootNodes = `-- name: CountRootNodes :one
SELECT COUNT(*)
FROM   PageNode
WHERE  depth = 0
    AND status_flags & ?1 = ?1
`

func (q *Queries) CountRootNodes(ctx context.Context, statusFlags page_models.StatusFlag) (int64, error) {
	row := q.queryRow(ctx, q.countRootNodesStmt, countRootNodes, statusFlags)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const deleteDescendants = `-- name: DeleteDescendants :exec
DELETE FROM PageNode
WHERE path LIKE CONCAT(?1, '%') AND depth > ?2
`

func (q *Queries) DeleteDescendants(ctx context.Context, path string, depth int64) error {
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
SELECT   id, title, path, depth, numchild, url_path, slug, status_flags, page_id, content_type, latest_revision_id, created_at, updated_at
FROM     PageNode
WHERE    path LIKE CONCAT(?1, '%')
    AND depth = ?2 + 1
    AND status_flags & ?3 = ?3
LIMIT    ?5
OFFSET   ?4
`

func (q *Queries) GetChildNodes(ctx context.Context, path string, depth int64, statusFlags page_models.StatusFlag, offset int32, limit int32) ([]page_models.PageNode, error) {
	rows, err := q.query(ctx, q.getChildNodesStmt, getChildNodes,
		path,
		depth,
		statusFlags,
		offset,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []page_models.PageNode
	for rows.Next() {
		var i page_models.PageNode
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

const getDescendants = `-- name: GetDescendants :many
SELECT   id, title, path, depth, numchild, url_path, slug, status_flags, page_id, content_type, latest_revision_id, created_at, updated_at
FROM     PageNode
WHERE    path LIKE CONCAT(?1, '%')
    AND depth > ?2
    AND status_flags & ?3 = ?3
LIMIT    ?5
OFFSET   ?4
`

func (q *Queries) GetDescendants(ctx context.Context, path string, depth int64, statusFlags page_models.StatusFlag, offset int32, limit int32) ([]page_models.PageNode, error) {
	rows, err := q.query(ctx, q.getDescendantsStmt, getDescendants,
		path,
		depth,
		statusFlags,
		offset,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []page_models.PageNode
	for rows.Next() {
		var i page_models.PageNode
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

const getNodeByID = `-- name: GetNodeByID :one
SELECT   id, title, path, depth, numchild, url_path, slug, status_flags, page_id, content_type, latest_revision_id, created_at, updated_at
FROM     PageNode
WHERE    id = ?1
`

func (q *Queries) GetNodeByID(ctx context.Context, id int64) (page_models.PageNode, error) {
	row := q.queryRow(ctx, q.getNodeByIDStmt, getNodeByID, id)
	var i page_models.PageNode
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

const getNodeByPath = `-- name: GetNodeByPath :one
SELECT   id, title, path, depth, numchild, url_path, slug, status_flags, page_id, content_type, latest_revision_id, created_at, updated_at
FROM     PageNode
WHERE    path = ?1
`

func (q *Queries) GetNodeByPath(ctx context.Context, path string) (page_models.PageNode, error) {
	row := q.queryRow(ctx, q.getNodeByPathStmt, getNodeByPath, path)
	var i page_models.PageNode
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

const getNodeBySlug = `-- name: GetNodeBySlug :one
SELECT   id, title, path, depth, numchild, url_path, slug, status_flags, page_id, content_type, latest_revision_id, created_at, updated_at
FROM     PageNode
WHERE    LOWER(slug) = LOWER(?1)
AND      depth =    ?2
AND      path  LIKE CONCAT(?3, '%')
`

func (q *Queries) GetNodeBySlug(ctx context.Context, slug string, depth int64, path string) (page_models.PageNode, error) {
	row := q.queryRow(ctx, q.getNodeBySlugStmt, getNodeBySlug, slug, depth, path)
	var i page_models.PageNode
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

const getNodesByDepth = `-- name: GetNodesByDepth :many
SELECT   id, title, path, depth, numchild, url_path, slug, status_flags, page_id, content_type, latest_revision_id, created_at, updated_at
FROM     PageNode
WHERE    depth = ?1
    AND status_flags & ?2 = ?2
LIMIT    ?4
OFFSET   ?3
`

func (q *Queries) GetNodesByDepth(ctx context.Context, depth int64, statusFlags page_models.StatusFlag, offset int32, limit int32) ([]page_models.PageNode, error) {
	rows, err := q.query(ctx, q.getNodesByDepthStmt, getNodesByDepth,
		depth,
		statusFlags,
		offset,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []page_models.PageNode
	for rows.Next() {
		var i page_models.PageNode
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

const getNodesByIDs = `-- name: GetNodesByIDs :many
SELECT   id, title, path, depth, numchild, url_path, slug, status_flags, page_id, content_type, latest_revision_id, created_at, updated_at
FROM     PageNode
WHERE    id IN (/*SLICE:id*/?)
`

func (q *Queries) GetNodesByIDs(ctx context.Context, id []int64) ([]page_models.PageNode, error) {
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
	var items []page_models.PageNode
	for rows.Next() {
		var i page_models.PageNode
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

const getNodesByPageIDs = `-- name: GetNodesByPageIDs :many
SELECT   id, title, path, depth, numchild, url_path, slug, status_flags, page_id, content_type, latest_revision_id, created_at, updated_at
FROM     PageNode
WHERE    page_id IN (/*SLICE:page_id*/?)
`

func (q *Queries) GetNodesByPageIDs(ctx context.Context, pageID []int64) ([]page_models.PageNode, error) {
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
	var items []page_models.PageNode
	for rows.Next() {
		var i page_models.PageNode
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

const getNodesByTypeHash = `-- name: GetNodesByTypeHash :many
SELECT   id, title, path, depth, numchild, url_path, slug, status_flags, page_id, content_type, latest_revision_id, created_at, updated_at
FROM     PageNode
WHERE    content_type = ?1
LIMIT    ?3
OFFSET   ?2
`

func (q *Queries) GetNodesByTypeHash(ctx context.Context, contentType string, offset int32, limit int32) ([]page_models.PageNode, error) {
	rows, err := q.query(ctx, q.getNodesByTypeHashStmt, getNodesByTypeHash, contentType, offset, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []page_models.PageNode
	for rows.Next() {
		var i page_models.PageNode
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

const getNodesByTypeHashes = `-- name: GetNodesByTypeHashes :many
SELECT   id, title, path, depth, numchild, url_path, slug, status_flags, page_id, content_type, latest_revision_id, created_at, updated_at
FROM     PageNode
WHERE    content_type IN (/*SLICE:content_type*/?)
LIMIT    ?3
OFFSET   ?2
`

func (q *Queries) GetNodesByTypeHashes(ctx context.Context, contentType []string, offset int32, limit int32) ([]page_models.PageNode, error) {
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
	queryParams = append(queryParams, offset)
	queryParams = append(queryParams, limit)
	rows, err := q.query(ctx, nil, query, queryParams...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []page_models.PageNode
	for rows.Next() {
		var i page_models.PageNode
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

const getNodesForPaths = `-- name: GetNodesForPaths :many
SELECT   id, title, path, depth, numchild, url_path, slug, status_flags, page_id, content_type, latest_revision_id, created_at, updated_at
FROM     PageNode
WHERE    path IN (/*SLICE:path*/?)
`

func (q *Queries) GetNodesForPaths(ctx context.Context, path []string) ([]page_models.PageNode, error) {
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
	var items []page_models.PageNode
	for rows.Next() {
		var i page_models.PageNode
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

const insertNode = `-- name: InsertNode :execlastid
INSERT INTO PageNode (
    title,
    path,
    depth,
    numchild,
    url_path,
    slug,
    status_flags,
    page_id,
    content_type,
    latest_revision_id
) VALUES (
    ?1,
    ?2,
    ?3,
    ?4,
    ?5,
    ?6,
    ?7,
    ?8,
    ?9,
    ?10
)
`

func (q *Queries) InsertNode(ctx context.Context, title string, path string, depth int64, numchild int64, urlPath string, slug string, statusFlags int64, pageID int64, contentType string, latestRevisionID int64) (int64, error) {
	result, err := q.exec(ctx, q.insertNodeStmt, insertNode,
		title,
		path,
		depth,
		numchild,
		urlPath,
		slug,
		statusFlags,
		pageID,
		contentType,
		latestRevisionID,
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
    slug = ?6,
    status_flags = ?7, 
    page_id = ?8, 
    content_type = ?9,
    latest_revision_id = ?10,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?11
`

func (q *Queries) UpdateNode(ctx context.Context, title string, path string, depth int64, numchild int64, urlPath string, slug string, statusFlags int64, pageID int64, contentType string, latestRevisionID int64, iD int64) error {
	_, err := q.exec(ctx, q.updateNodeStmt, updateNode,
		title,
		path,
		depth,
		numchild,
		urlPath,
		slug,
		statusFlags,
		pageID,
		contentType,
		latestRevisionID,
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
