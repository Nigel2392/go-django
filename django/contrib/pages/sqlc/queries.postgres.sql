-- name: InsertNode :execlastid
INSERT INTO PageNode (title, path, depth, numchild, page_id, typeHash)
VALUES (sqlc.arg(title), sqlc.arg(path), sqlc.arg(depth), sqlc.arg(numchild), sqlc.arg(page_id), sqlc.arg(typeHash));

-- name: GetNodeByID :one
SELECT id, title, path, depth, numchild, page_id, typeHash
FROM PageNode
WHERE id = sqlc.arg(id);

-- name: GetNodeByPath :one
SELECT id, title, path, depth, numchild, page_id, typeHash
FROM PageNode
WHERE path = sqlc.arg(path);

-- name: GetChildren :many
SELECT id, title, path, depth, numchild, page_id, typeHash
FROM PageNode
WHERE path LIKE CONCAT(sqlc.arg(path), '%') AND depth = sqlc.arg(depth) + 1;

-- name: GetDescendants :many
SELECT id, title, path, depth, numchild, page_id, typeHash
FROM PageNode
WHERE path LIKE CONCAT(sqlc.arg(path), '%') AND depth > sqlc.arg(depth);

-- name: UpdateNode :exec
UPDATE PageNode
SET title = sqlc.arg(title), path = sqlc.arg(path), depth = sqlc.arg(depth), numchild = sqlc.arg(numchild), page_id = sqlc.arg(page_id), typeHash = sqlc.arg(typeHash)
WHERE id = sqlc.arg(id);

-- name: DeleteNode :exec
DELETE FROM PageNode
WHERE id = sqlc.arg(id);

-- name: UpdateNodePathAndDepth :exec
UPDATE PageNode
SET path = sqlc.arg(path), depth = sqlc.arg(depth)
WHERE id = sqlc.arg(id);
