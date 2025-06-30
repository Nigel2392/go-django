-- name: InsertImage :execlastid
INSERT INTO images (
    title,
    path,
    created_at,
    file_size,
    file_hash
) VALUES (
    sqlc.arg(title),
    sqlc.arg(path),
    sqlc.arg(created_at),
    sqlc.arg(file_size),
    sqlc.arg(file_hash)
);

-- name: UpdateImage :exec
UPDATE images
SET
    title = sqlc.arg(title),
    path = sqlc.arg(path),
    created_at = sqlc.arg(created_at),
    file_size = sqlc.arg(file_size),
    file_hash = sqlc.arg(file_hash)
WHERE id = sqlc.arg(id);

-- name: DeleteImage :exec
DELETE FROM images
WHERE id = sqlc.arg(id);

-- name: SelectByID :one
SELECT *
FROM images
WHERE id = sqlc.arg(id);

-- name: SelectByPath :one
SELECT *
FROM images
WHERE path = sqlc.arg(path);

-- name: SelectBasic :many
SELECT *
FROM images
ORDER BY id
LIMIT ? OFFSET ?;

-- name: SelectLargeToSmall :many
SELECT *
FROM images
ORDER BY file_size DESC
LIMIT ? OFFSET ?;

-- name: SelectSmallToLarge :many
SELECT *
FROM images
ORDER BY file_size
LIMIT ? OFFSET ?;

-- name: SelectNewestToOldest :many
SELECT *
FROM images
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: SelectOldestToNewest :many
SELECT *
FROM images
ORDER BY created_at
LIMIT ? OFFSET ?;

-- name: SelectByFileHash :one
SELECT *
FROM images
WHERE file_hash = sqlc.arg(file_hash);
