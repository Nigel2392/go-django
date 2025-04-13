-- name: RetrieveUserByID :one
SELECT * FROM oauth2_users
WHERE id = sqlc.arg(id);

-- name: RetrieveUserByIdentifier :one
SELECT * FROM oauth2_users
WHERE unique_identifier = sqlc.arg(unique_identifier);

-- name: CreateUser :execlastid
INSERT INTO oauth2_users (unique_identifier, data, is_administrator, is_active)
VALUES (
    sqlc.arg(unique_identifier),
    sqlc.arg(data),
    sqlc.arg(is_administrator),
    sqlc.arg(is_active)
);

-- name: UpdateUser :exec
UPDATE oauth2_users
SET unique_identifier = sqlc.arg(unique_identifier),
    data = sqlc.arg(data),
    is_administrator = sqlc.arg(is_administrator),
    is_active = sqlc.arg(is_active)
WHERE id = sqlc.arg(id);

-- name: DeleteUser :exec
DELETE FROM oauth2_users
WHERE id = sqlc.arg(id);

-- name: DeleteUsers :exec
DELETE FROM oauth2_users
WHERE id IN (sqlc.slice(ids));