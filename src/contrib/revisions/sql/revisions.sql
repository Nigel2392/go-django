-- name: InsertRevision :execlastid
INSERT INTO Revision (
    object_id,
    content_type,
    data
) VALUES (
    sqlc.arg(object_id),
    sqlc.arg(content_type),
    sqlc.arg(data)
);

-- name: GetRevisionByID :one
SELECT   *
FROM     Revision
WHERE    id = sqlc.arg(id);

-- name: GetRevisionsByObjectID :many
SELECT   *
FROM     Revision
WHERE    object_id = ?
AND      content_type = ?
ORDER BY created_at DESC
LIMIT    ?
OFFSET   ?;

-- name: ListRevisions :many
SELECT   *
FROM     Revision
ORDER BY created_at DESC
LIMIT    ?
OFFSET   ?;

-- name: UpdateRevision :exec
UPDATE Revision
SET object_id = sqlc.arg(object_id),
    content_type = sqlc.arg(content_type),
    data = sqlc.arg(data)
WHERE id = sqlc.arg(id);

-- name: DeleteRevision :exec
DELETE FROM Revision
WHERE id = sqlc.arg(id);
