-- name: InsertNode :execlastid
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
    sqlc.arg(title),
    sqlc.arg(path),
    sqlc.arg(depth),
    sqlc.arg(numchild),
    sqlc.arg(url_path),
    sqlc.arg(slug),
    sqlc.arg(status_flags),
    sqlc.arg(page_id),
    sqlc.arg(content_type),
    sqlc.arg(latest_revision_id)
);

-- name: AllNodes :many
SELECT *
FROM     PageNode
WHERE    status_flags & sqlc.arg(status_flags) = sqlc.arg(status_flags)
ORDER BY path ASC
LIMIT    sqlc.arg(limit)
OFFSET   sqlc.arg(offset);

-- name: CountRootNodes :one
SELECT COUNT(*)
FROM   PageNode
WHERE  depth = 0
    AND status_flags & sqlc.arg(status_flags) = sqlc.arg(status_flags);

-- name: CountNodes :one
SELECT COUNT(*)
FROM   PageNode
WHERE status_flags & sqlc.arg(status_flags) = sqlc.arg(status_flags);

-- name: CountNodesByTypeHash :one
SELECT COUNT(*)
FROM   PageNode
WHERE  content_type = sqlc.arg(content_type);

-- name: GetNodeByID :one
SELECT   *
FROM     PageNode
WHERE    id = sqlc.arg(id);

-- name: GetNodeBySlug :one
SELECT   *
FROM     PageNode
WHERE    LOWER(slug) = LOWER(sqlc.arg(slug))
AND      depth =    sqlc.arg(depth)
AND      path  LIKE CONCAT(sqlc.arg(path), '%');

-- name: GetNodesByIDs :many
SELECT   *
FROM     PageNode
WHERE    id IN (sqlc.slice(id));

-- name: GetNodesByDepth :many
SELECT   *
FROM     PageNode
WHERE    depth = sqlc.arg(depth)
    AND status_flags & sqlc.arg(status_flags) = sqlc.arg(status_flags)
LIMIT    sqlc.arg(limit)
OFFSET   sqlc.arg(offset);

-- name: GetNodesByPageIDs :many
SELECT   *
FROM     PageNode
WHERE    page_id IN (sqlc.slice(page_id));

-- name: GetNodesByTypeHash :many
SELECT   *
FROM     PageNode
WHERE    content_type = sqlc.arg(content_type)
LIMIT    sqlc.arg(limit)
OFFSET   sqlc.arg(offset);

-- name: GetNodesByTypeHashes :many
SELECT   *
FROM     PageNode
WHERE    content_type IN (sqlc.slice(content_type))
LIMIT    sqlc.arg(limit)
OFFSET   sqlc.arg(offset);

-- name: GetNodeByPath :one
SELECT   *
FROM     PageNode
WHERE    path = sqlc.arg(path);

-- name: GetNodesForPaths :many
SELECT   *
FROM     PageNode
WHERE    path IN (sqlc.slice(path));

-- name: GetChildNodes :many
SELECT   *
FROM     PageNode
WHERE    path LIKE CONCAT(sqlc.arg(path), '%')
    AND depth = sqlc.arg(depth) + 1
    AND status_flags & sqlc.arg(status_flags) = sqlc.arg(status_flags)
LIMIT    sqlc.arg(limit)
OFFSET   sqlc.arg(offset);

-- name: GetDescendants :many
SELECT   *
FROM     PageNode
WHERE    path LIKE CONCAT(sqlc.arg(path), '%')
    AND depth > sqlc.arg(depth)
    AND status_flags & sqlc.arg(status_flags) = sqlc.arg(status_flags)
LIMIT    sqlc.arg(limit)
OFFSET   sqlc.arg(offset);

-- name: UpdateNode :exec
UPDATE PageNode
SET title = sqlc.arg(title),
    path = sqlc.arg(path),
    depth = sqlc.arg(depth), 
    numchild = sqlc.arg(numchild),
    url_path = sqlc.arg(url_path),
    slug = sqlc.arg(slug),
    status_flags = sqlc.arg(status_flags), 
    page_id = sqlc.arg(page_id), 
    content_type = sqlc.arg(content_type),
    latest_revision_id = sqlc.arg(latest_revision_id),
    updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id);

-- name: UpdateNodePathAndDepth :exec
UPDATE   PageNode
SET      path = sqlc.arg(path), depth = sqlc.arg(depth)
WHERE    id = sqlc.arg(id);

-- name: UpdateNodeStatusFlags :exec
UPDATE   PageNode
SET      status_flags = sqlc.arg(status_flags)
WHERE    id = sqlc.arg(id);

-- name: IncrementNumChild :exec
UPDATE PageNode
SET numchild = numchild + 1
WHERE id = sqlc.arg(id);

-- name: DecrementNumChild :exec
UPDATE PageNode
SET numchild = numchild - 1
WHERE id = sqlc.arg(id);

-- name: DeleteNode :exec
DELETE FROM PageNode
WHERE id = sqlc.arg(id);

-- name: DeleteNodes :exec
DELETE FROM PageNode
WHERE id IN (sqlc.slice(id));

-- name: DeleteDescendants :exec
DELETE FROM PageNode
WHERE path LIKE CONCAT(sqlc.arg(path), '%') AND depth > sqlc.arg(depth);