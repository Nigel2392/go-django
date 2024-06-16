-- name: InsertNode :execlastid
INSERT INTO PageNode (
    title,
    path,
    depth,
    numchild,
    status_flags,
    page_id,
    typeHash
) VALUES (
    sqlc.arg(title),
    sqlc.arg(path),
    sqlc.arg(depth),
    sqlc.arg(numchild),
    sqlc.arg(status_flags),
    sqlc.arg(page_id),
    sqlc.arg(typeHash)
);

-- name: GetNodeByID :one
SELECT *
FROM PageNode
WHERE id = sqlc.arg(id);

-- name: GetNodesByIDs :many
SELECT *
FROM PageNode
WHERE id IN (sqlc.slice(id));

-- name: GetNodesByPageIDs :many
SELECT *
FROM PageNode
WHERE page_id IN (sqlc.slice(page_id));

-- name: GetNodesByTypeHash :many
SELECT *
FROM PageNode
WHERE typeHash = sqlc.arg(typeHash);

-- name: GetNodesByTypeHashes :many
SELECT *
FROM PageNode
WHERE typeHash IN (sqlc.slice(typeHash));

-- name: GetNodeByPath :one
SELECT *
FROM PageNode
WHERE path = sqlc.arg(path);

-- name: GetNodeForPath :many
SELECT *
FROM PageNode
WHERE path IN (sqlc.slice(path));

-- name: GetChildNodes :many
SELECT *
FROM PageNode
WHERE path LIKE CONCAT(sqlc.arg(path), '%') AND depth = sqlc.arg(depth) + 1;

-- name: GetDescendants :many
SELECT *
FROM PageNode
WHERE path LIKE CONCAT(sqlc.arg(path), '%') AND depth > sqlc.arg(depth);

-- name: UpdateNode :exec
UPDATE PageNode
SET title = sqlc.arg(title),
    path = sqlc.arg(path),
    depth = sqlc.arg(depth), 
    numchild = sqlc.arg(numchild), 
    status_flags = sqlc.arg(status_flags), 
    page_id = sqlc.arg(page_id), 
    typeHash = sqlc.arg(typeHash)
WHERE id = sqlc.arg(id);

-- name: UpdateNodePathAndDepth :exec
UPDATE PageNode
SET path = sqlc.arg(path), depth = sqlc.arg(depth)
WHERE id = sqlc.arg(id);

-- name: UpdateNodeStatusFlags :exec
UPDATE PageNode
SET status_flags = sqlc.arg(status_flags)
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
